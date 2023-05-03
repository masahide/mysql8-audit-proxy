package generatepem

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"
)

type Config struct {
	Host     string `envconfig:"HOST" required:"true" description:"Comma-separated hostnames and IPs to generate a certificate for"`
	ValidFor int    `envconfig:"TIME_DATE" default:"10000" description:"expiry time in days"`
	//IsCA       bool   `envconfig:"CA" default:"false" description:"whether this cert should be its own Certificate Authority"`
	RsaBits    int    `envconfig:"RSA_BITS" default:"2048" description:"Size of RSA key to generate. Ignored if --ecdsa-curve is set"`
	EcdsaCurve string `envconfig:"ECDSA_CURVE" default:"" description:"ECDSA curve to use to generate a key. Valid values are P224, P256 (recommended), P384, P521"`
	Ed25519Key bool   `envconfig:"ED25519" default:"false" description:"Generate an Ed25519 key"`
}

type Pems struct {
	Cert   string
	Key    string
	Public string
}

func Generate(c Config) (Pems, Pems, error) {
	var priv any
	var err error
	switch c.EcdsaCurve {
	case "":
		if c.Ed25519Key {
			_, priv, err = ed25519.GenerateKey(rand.Reader)
		} else {
			priv, err = rsa.GenerateKey(rand.Reader, c.RsaBits)
		}
	case "P224":
		priv, err = ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
	case "P256":
		priv, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case "P384":
		priv, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	case "P521":
		priv, err = ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	default:
		return Pems{}, Pems{}, fmt.Errorf("unrecognized elliptic curve: %q", c.EcdsaCurve)
	}
	if err != nil {
		return Pems{}, Pems{}, fmt.Errorf("failed to generate private key: err: %w", err)
	}

	// ECDSA, ED25519 and RSA subject keys should have the DigitalSignature
	// KeyUsage bits set in the x509.Certificate template
	keyUsage := x509.KeyUsageDigitalSignature
	// Only RSA subject keys should have the KeyEncipherment KeyUsage bits set. In
	// the context of TLS this KeyUsage is particular to RSA key exchange and
	// authentication.
	if _, isRSA := priv.(*rsa.PrivateKey); isRSA {
		keyUsage |= x509.KeyUsageKeyEncipherment
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Duration(c.ValidFor) * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return Pems{}, Pems{}, fmt.Errorf("failed to generate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              keyUsage,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	hosts := strings.Split(c.Host, ",")
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	caTemplate := template
	caTemplate.IsCA = true
	caTemplate.KeyUsage = keyUsage | x509.KeyUsageCertSign
	caPems, err := generatePems(caTemplate, caTemplate, priv)
	if err != nil {
		return Pems{}, Pems{}, err
	}
	serverPems, err := generatePems(template, caTemplate, priv)
	return caPems, serverPems, err

}

func generatePems(template, caTemplate x509.Certificate, priv any) (Pems, error) {
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &caTemplate, publicKey(priv), priv)
	if err != nil {
		return Pems{}, fmt.Errorf("failed to create certificate: %w", err)
	}

	certOut := &bytes.Buffer{}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return Pems{}, fmt.Errorf("failed to write data to cert.pem: %w", err)
	}
	keyOut := &bytes.Buffer{}
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return Pems{}, fmt.Errorf("unable to marshal private key: %w", err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		return Pems{}, fmt.Errorf("failed to write data to key.pem: %w", err)
	}

	// dump public key to file
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey(priv))
	if err != nil {
		return Pems{}, fmt.Errorf("error when dumping publickey: %w", err)
	}
	publicKeyBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}
	publicPem := &bytes.Buffer{}
	if err = pem.Encode(publicPem, publicKeyBlock); err != nil {
		return Pems{}, fmt.Errorf("error when encode public pem: %w", err)
	}
	return Pems{
		Cert:   certOut.String(),
		Key:    keyOut.String(),
		Public: publicPem.String(),
	}, nil
}

func publicKey(priv any) any {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	case ed25519.PrivateKey:
		return k.Public().(ed25519.PublicKey)
	default:
		return nil
	}
}
