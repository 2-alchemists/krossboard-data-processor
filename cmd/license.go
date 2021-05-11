package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/hyperboloide/lk"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"strings"
	"time"
)

type AppLicense struct {
	MajorVersion    string
	IssuedAt        time.Time
	ExpireAt        time.Time
	Issuer          string
	ExtendedSupport bool
}

const KrossboardLicenseTokenConfigKey = "krossboard_license_token"
const KrossboardLicensePrivKeyConfigKey = "krossboard_license_priv_key"
const KrossboardLicensePubKeyConfigKey = "krossboard_license_pub_key"


func createLicenseKeyPair() (privKeyB64 string, pubKeyB64 string, err error){
	privKey, err := lk.NewPrivateKey()
	if err != nil {
		return "", "", errors.Wrap(err, "failed creating private key")
	}

	privKeyB64, err = privKey.ToB64String()
	if err != nil {
		return "", "", errors.Wrap(err, "failed getting the private key in base64")
	}

	pubKey := privKey.GetPublicKey()
	pubKeyB64 = pubKey.ToB64String()
	if err != nil {
		return "", "", errors.Wrap(err, "failed getting the public key in base64")
	}

	return privKeyB64, pubKeyB64, nil
}

func createLicenseTokenFromEnvConfig(appVersion string, duration time.Duration) (licenseB64 string, err error){
	semVerParts := strings.Split(appVersion, ".")
	if len(semVerParts) != 3 {
		return "", errors.New("unexpected version string => " + appVersion)
	}

	licenseIssueDate := time.Now()
	licenseDoc := AppLicense{
		MajorVersion:    fmt.Sprintf("%s.%s", semVerParts[0], semVerParts[1]),
		IssuedAt:        licenseIssueDate,
		ExpireAt:        licenseIssueDate.Add(duration),
		Issuer:          "2Alchemists SAS",
		ExtendedSupport: false,
	}

	// marshall the license document to json:
	licenseDocEncoded, err := json.Marshal(licenseDoc)
	if err != nil {
		return "", errors.Wrap(err, "failed encoding JSON license document")
	}

	privKeyB64 := viper.GetString(KrossboardLicensePrivKeyConfigKey)
	privKey, err := lk.PrivateKeyFromB64String(privKeyB64)
	if err != nil {
		return "", errors.Wrap(err, "base64-decoding of the private key failed")
	}

	license, err := lk.NewLicense(privKey, licenseDocEncoded)
	if err != nil {
		return "", errors.Wrap(err, "failed generating a license")
	}

	// encode the new license to b64, this is what you give to your customer.
	licenseB64, err = license.ToB64String()
	if err != nil {
		return "", errors.Wrap(err, "failed encoding the license in base64")
	}
	return licenseB64, nil
}

func validateLicenseFromEnvConfig(version string) (licenseDoc *AppLicense, err error) {
	licenseB64 := viper.GetString(KrossboardLicenseTokenConfigKey)
	license, err := lk.LicenseFromB64String(licenseB64)
	if err != nil {
		return nil, errors.Wrap(err, "failed decoding license token")
	}

	licensePubKeyB64 := viper.GetString(KrossboardLicensePubKeyConfigKey)
	licensePubKey, err := lk.PublicKeyFromB64String(licensePubKeyB64)
	if err != nil {
		return nil, errors.Wrap(err, "failed decoding license public key")
	}

	if ok, err := license.Verify(licensePubKey); err != nil {
		return nil, errors.Wrap(err, "failed verify the license signature")
	} else if !ok {
		return nil, errors.New("invalid license signature")
	}

	licenseDoc = &AppLicense{}
	if err := json.Unmarshal(license.Data, licenseDoc); err != nil {
		return nil, errors.Wrap(err, "failed decoding in JSON")
	}

	if licenseDoc.ExpireAt.Before(time.Now()) {
		return nil, errors.New(fmt.Sprintf(
			"license issued at %v expired at %v", licenseDoc.IssuedAt,licenseDoc.ExpireAt))
	}

	semVerParts := strings.Split(version, ".")
	if len(semVerParts) != 3 {
		return nil, errors.New("unexpected version string => " + version)
	}
	majorVersion := fmt.Sprintf("%s.%s", semVerParts[0], semVerParts[1])
	if licenseDoc.MajorVersion != majorVersion {
		return nil, errors.New(fmt.Sprintf("the license is valid with the major version %v (current version: %v)", licenseDoc.MajorVersion, version))
	}

	return licenseDoc,nil
}