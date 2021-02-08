package cmd

import (
	"fmt"
	"github.com/hyperboloide/lk"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/spf13/viper"
	"testing"
)

func TestKeyPairGeneration(t *testing.T) {
	Convey("Test license key pair management", t, func() {

		privKeyB64, pubKeyB64, err := createLicenseKeyPair()
		Convey("creation should succeed", func() {
			So(err, ShouldBeNil)
		})

		Convey("the private key should not be empty", func() {
			So(privKeyB64, ShouldNotBeEmpty)
		})

		Convey("the public key should not be empty", func() {
			So(pubKeyB64, ShouldNotBeEmpty)
		})

		Convey("base64-decoding of the private key should succeed", func() {
			_, err = lk.PrivateKeyFromB64String(privKeyB64)
			So(err, ShouldBeNil)
		})

		Convey("base64-decoding of the public key should succeed", func() {
			_, err = lk.PublicKeyFromB64String(pubKeyB64)
			So(err, ShouldBeNil)
		})
	})
}


func TestCreateLicenseToken(t *testing.T) {
	Convey("Test create license token", t, func() {

		privKeyB64, pubKeyB64, err := createLicenseKeyPair()
		Convey("key pair creation should succeed", func() {
			So(err, ShouldBeNil)
			So(privKeyB64, ShouldNotBeEmpty)
			So(pubKeyB64, ShouldNotBeEmpty)
		})

		viper.Set(KrossboardLicensePrivKeyConfigKey, privKeyB64)

		licenseB64, err := createLicenseTokenFromEnv(KrossboardVersion)
		Convey("license creation should succeed", func() {
			So(err, ShouldBeNil)
			So(licenseB64, ShouldNotBeEmpty)
		})

		Convey("base64-decoding of the license should succeed", func() {
			_, err = lk.PublicKeyFromB64String(pubKeyB64)
			So(err, ShouldBeNil)
		})


		Convey("the created license should be valid", func() {
			viper.Set(KrossboardLicenseTokenConfigKey, licenseB64)
			viper.Set(KrossboardLicensePubKeyConfigKey, pubKeyB64)
			licenseDoc, err := validateLicenseFromEnv()
			So(err, ShouldBeNil)
			fmt.Println(licenseDoc)
		})
	})
}