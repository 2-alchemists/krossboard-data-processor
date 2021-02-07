package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/hyperboloide/lk"
	"github.com/spf13/viper"
	"os"
	"strings"
	"time"
)

type AppLicense struct {
	Version string
	GeneratedAt time.Time
	ExpireAt time.Time
	Issuer string
	ExtendedSupport bool
}

const LicenseTokenConfigKey = "krossboard_license_token"
const LicensePrivKeyConfigKey = "krossboard_license_priv_key"
const LicensePubKeyConfigKey = "krossboard_license_pub_key"


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

	pubKey := privKey.GetPublicKey()
	pubKey64 := pubKey.ToB64String()
	if err != nil {
		fmt.Println("✘ Failed getting the public key in base64:", err)
		os.Exit(1)
	}

	fmt.Println("✓ Success")
	fmt.Printf("%s=%s\n", strings.ToUpper(LicensePrivKeyConfigKey), privKey64)
	fmt.Printf("%s=%s\n", strings.ToUpper(LicensePubKeyConfigKey), pubKey64)
}

func createAppVersionLicense() {
	semVerParts := strings.Split(licenseTargetVersion, ".")
	if len(semVerParts) != 3 {
		fmt.Println("✘ Unsupported target license version. Please provide a valid license version (e.g. --target-version=1.1.0)")
		os.Exit(1)
	}

	licenseIssueDate := time.Now()
	licenseDoc := AppLicense{
		Version: fmt.Sprintf("%s.%s"),
		GeneratedAt: licenseIssueDate,
		ExpireAt: licenseIssueDate.Add(time.Hour * 24 * 365), // one-year license
		Issuer: "2Alchemists SAS",
		ExtendedSupport: false,
	}

	// marshall the license document to json:
	docBytes, err := json.Marshal(licenseDoc)
	if err != nil {
		fmt.Println("✘ Failed encoding JSON license document", err)
		os.Exit(1)
	}

	privKeyB64 := viper.GetString(LicensePrivKeyConfigKey)
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
	fmt.Println("✓ Success")
	fmt.Printf("%s=%s\n", strings.ToUpper(LicenseTokenConfigKey), licenseB64)
}
