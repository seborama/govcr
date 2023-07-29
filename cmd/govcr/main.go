package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/pkg/errors"

	"github.com/seborama/govcr/v14/cassette"
	"github.com/seborama/govcr/v14/encryption"
)

func main() {
	decryptCmd := flag.NewFlagSet("decrypt", flag.ExitOnError)

	cassetteFile := decryptCmd.String("cassette-file", "", "location of the cassette file to decrypt")
	keyFile := decryptCmd.String("key-file", "", "location of the encryption key file")

	if len(os.Args) < 2 {
		help()
		os.Exit(100)
	}

	switch os.Args[1] {
	case "decrypt":
		if err := decryptCmd.Parse(os.Args[2:]); err != nil {
			fmt.Println(err)
			os.Exit(100)
		}

		if err := decryptCommand(*cassetteFile, *keyFile); err != nil {
			fmt.Println(err)
			os.Exit(100)
		}

	default:
		help()
		os.Exit(100)
	}
}

func help() {
	fmt.Println(`please specify a sub-command:
   decrypt: decrypts an encrypted cassette to the standard output.`)
}

func decryptCommand(cassetteFile, keyFile string) error {
	if cassetteFile == "" {
		return errors.New("please specify a cassette file with the 'cassette-file' argument")
	}

	if keyFile == "" {
		return errors.New("please specify a key file with the 'key-file' argument")
	}

	data, err := decryptCassette(cassetteFile, keyFile)
	if err != nil {
		return err
	}

	fmt.Println(data)
	return nil
}

// TODO: offer ability to supply the key via an environment variable in base64 format.
func decryptCassette(cassetteFile, keyFile string) (string, error) {
	key, err := os.ReadFile(keyFile)
	if err != nil {
		return "", errors.Wrap(err, "key file")
	}

	crypter, err := encryption.NewAESGCMWithRandomNonceGenerator(key)
	if err != nil {
		return "", errors.Wrap(err, "cryptographer")
	}

	data := cassette.DumpCassette(cassetteFile, cassette.WithCrypter(crypter))

	return string(data), nil
}
