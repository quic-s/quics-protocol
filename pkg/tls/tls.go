package tls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"log"
	"math/big"
	"os"
)

func GetCertificate(keyPath string, certPath string) ([]tls.Certificate, error) {
	_, keyErr := os.Stat(keyPath)
	_, certErr := os.Stat(certPath)
	cert, err := &tls.Certificate{}, error(nil)
	if os.IsNotExist(keyErr) && os.IsNotExist(certErr) {
		log.Println("quics-protocol: ", "generate ssl")
		cert, err = GenerateSSL()
		if err != nil {
			return nil, err
		}
	} else {
		cert, err = GetSSL(keyPath, certPath)
		if err != nil {
			return nil, err
		}
	}

	return []tls.Certificate{*cert}, nil
}

func GenerateSSL() (*tls.Certificate, error) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, err
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	keyOut := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certOut := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	cert, err := tls.X509KeyPair(certOut, keyOut)
	if err != nil {
		return nil, err
	}

	return &cert, nil
}

func GetSSL(keyPath string, certPath string) (*tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}

	return &cert, nil
}
