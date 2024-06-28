package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-steputils/tools"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
)

// Define variable
var isError bool = false

// Config ...
type Config struct {
	GMCloudSaaSEmail         string          `env:"email"`
	GMCloudSaaSPassword      stepconf.Secret `env:"password"`
	GMCloudSaaSGmsaasVersion string          `env:"gmsaas_version"`
	GMCloudSaaSAPIToken      stepconf.Secret `env:"api_token"`
}

// install gmsaas if not installed.
func ensureGMSAASisInstalled(version string) error {
	path, err := exec.LookPath("gmsaas")
	if err != nil {
		log.Infof("Installing gmsaas...")

		var installCmd *exec.Cmd
		if version != "" {
			installCmd = exec.Command("pip3", "install", "gmsaas=="+version, "--break-system-packages")
		} else {
			installCmd = exec.Command("pip3", "install", "gmsaas", "--break-system-packages")
		}

		if out, err := installCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%s failed, error: %s | output: %s", installCmd.Args, err, out)
		}

		// Execute asdf reshim to update PATH
		exec.Command("asdf", "reshim", "python").CombinedOutput()

		if version != "" {
			log.Infof("gmsaas %s has been installed.", version)
		} else {
			log.Infof("gmsaas has been installed.")
		}

	} else {
		log.Infof("gmsaas is already installed: %s", path)
	}

	// Set Custom user agent to improve customer support
	os.Setenv("GMSAAS_USER_AGENT_EXTRA_DATA", "bitrise.io")
	return nil
}

// printError prints an error.
func printError(format string, args ...interface{}) {
	log.Errorf(format, args...)
}

// abortf prints an error and terminates step
func abortf(format string, args ...interface{}) {
	printError(format, args...)
	os.Exit(1)
}

// setOperationFailed marked step as failed
func setOperationFailed(format string, args ...interface{}) {
	printError(format, args...)
	isError = true
}

func configureAndroidSDKPath() {
	log.Infof("Configure Android SDK configuration")

	value, exists := os.LookupEnv("ANDROID_HOME")
	if exists {
		cmd := command.New("gmsaas", "config", "set", "android-sdk-path", value)
		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		if err != nil {
			setOperationFailed("Failed to set android-sdk-path, error: error: %s | output: %s", cmd.PrintableCommandArgs(), err, out)
			return
		}
		log.Infof("Android SDK is configured")
	} else {
		setOperationFailed("Please set ANDROID_HOME environment variable")
		return
	}
}

func login(api_token, username, password string) {
	log.Infof("Login Genymotion Account")

	var cmd *exec.Cmd
	if api_token != "" {
		cmd = exec.Command("gmsaas", "auth", "token", api_token)
	} else if username != "" && password != "" {
		cmd = exec.Command("gmsaas", "auth", "login", username, password)
	} else {
		abortf("Invalid arguments. Must provide either a token or both email and password.")
		return
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		abortf("Failed to login with gmsaas, error: error: %s | output: %s", cmd.Args, err, out)
		return
	}

	log.Infof("Logged to Genymotion Cloud SaaS platform")
}

func main() {

	var c Config
	if err := stepconf.Parse(&c); err != nil {
		abortf("Issue with input: %s", err)
	}
	stepconf.Print(c)

	if err := ensureGMSAASisInstalled(c.GMCloudSaaSGmsaasVersion); err != nil {
		abortf("%s", err)
	}
	configureAndroidSDKPath()

	if err := tools.ExportEnvironmentWithEnvman("GMSAAS_USER_AGENT_EXTRA_DATA", "bitrise.io"); err != nil {
		printError("Failed to export %s, error: %v", "GMSAAS_USER_AGENT_EXTRA_DATA", err)
	}

	if c.GMCloudSaaSAPIToken != "" {
		login(string(c.GMCloudSaaSAPIToken), "", "")
	} else {
		login("", c.GMCloudSaaSEmail, string(c.GMCloudSaaSPassword))
	}

	// --- Exit codes:
	// The exit code of your Step is very important. If you return
	//  with a 0 exit code `bitrise` will register your Step as "successful".
	// Any non zero exit code will be registered as "failed" by `bitrise`.
	if isError {
		// If at least one error happens, step will fail
		os.Exit(1)
	}
	os.Exit(0)
}
