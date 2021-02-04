package cmd

import (
	"fmt"
	"github.com/hyperboloide/lk"
	"github.com/spf13/viper"
	"os"
	"time"
	"encoding/json"
)

type AppLicense struct {
	Version string
	GeneratedAt time.Time
	ExpireAt time.Time
	Issuer string
	ExtendedSupport bool
}

func createLicenseKeyPair() {
	privKey, err := lk.NewPrivateKey()
	if err != nil {
		fmt.Println("✘ Failed creating private key:", err)
		os.Exit(1)
	}

	privKey64, err := privKey.ToB64String()
	if err != nil {
		fmt.Println("✘ Failed getting the private key in base64:", err)
		os.Exit(1)
	}

	fmt.Println("✓ Private Key =>", privKey64)

	pubKey := privKey.GetPublicKey()
	pubKey64 := pubKey.ToB64String()
	if err != nil {
		fmt.Println("✘ Failed getting the public key in base64:", err)
		os.Exit(1)
	}

	fmt.Println("✓ Public Key =>", pubKey64)
}

func createAppVersionLicense() {
	// create a license document:
	licenseIssueDate := time.Now()
	licenseDoc := AppLicense{
		Version: "1.1.0",
		GeneratedAt: licenseIssueDate,
		ExpireAt: licenseIssueDate.Add(time.Hour * 24 * 365),
		Issuer: "2Alchemists SAS",
		ExtendedSupport: true,
	}

	// marshall the document to json bytes:
	docBytes, err := json.Marshal(licenseDoc)
	if err != nil {
		fmt.Println("✘ Failed encoding JSON license document", err)
		os.Exit(1)
	}

	privKeyB64 := viper.GetString("krossboard_license_priv_key")
	privKey, err := lk.PrivateKeyFromB64String(privKeyB64)
	if err != nil {
		fmt.Println("✘ Failed decoding LK private key from base64 string", err)
		os.Exit(1)
	}

	license, err := lk.NewLicense(privKey, docBytes)
	if err != nil {
		fmt.Println("✘ Failed generating a license", err)
		os.Exit(1)
	}

	// encode the new license to b64, this is what you give to your customer.
	licenseB64, err := license.ToB64String()
	if err != nil {
		fmt.Println("✘ Failed encoding the license in base64", err)
		os.Exit(1)
	}
	fmt.Println("✓ License =>", licenseB64)
}
