package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/seungbemi/gofred"
	"gopkg.in/yaml.v2"
)

const (
	noSubtitle     = ""
	noArg          = ""
	noAutocomplete = ""
	configFolder   = "conf"
)

// Config includes
type Config struct {
	Proxy   string `yaml:"Proxy"`
	User    string `yaml:"User"`
	Cluster string `yaml:"Cluster"`
	Auth    string `yaml:"Auth"`
	Format  string `yaml:"Format"`
}

// Message adds simple message
func Message(response *gofred.Response, title, subtitle string, err bool) {
	msg := gofred.NewItem(title, subtitle, noAutocomplete)
	// if err {
	// 	msg = msg.AddIcon(iconError, defaultIconType)
	// } else {
	// 	msg = msg.AddIcon(iconDone, defaultIconType)
	// }
	response.AddItems(msg)
	fmt.Println(response)
}

func init() {
	flag.Parse()
}
func main() {
	path := os.Getenv("PATH")
	if !strings.Contains(path, "/usr/local/bin") {
		os.Setenv("PATH", path+":/usr/local/bin")
	}
	configPath := os.Getenv("alfred_workflow_data") + "/" + configFolder
	response := gofred.NewResponse()

	err := os.MkdirAll(configPath, os.ModePerm)
	if err != nil {
		Message(response, "error", err.Error(), true)
		return
	}

	configs, err := ioutil.ReadDir(configPath)
	if err != nil {
		Message(response, "error", err.Error(), true)
		return
	}
	items := []gofred.Item{}
	if flag.Arg(0) != "create" {
		for _, config := range configs {
			var teleport Config
			bt, err := ioutil.ReadFile(configPath + "/" + config.Name())
			if err != nil {
				Message(response, "error", err.Error(), true)
				return
			}
			if err = yaml.Unmarshal(bt, &teleport); err != nil {
				Message(response, "error", err.Error(), true)
				return
			}
			_ = teleport.isValid()

			name := strings.TrimSuffix(config.Name(), ".yml")
			baseCMD := teleport.baseCMD()

			status := "Off"
			cmd := "Login"
			checkerCMD := fmt.Sprintf("%s clusters -q", baseCMD)
			shellCMD := fmt.Sprintf(`
				PWD=$(osascript -e 'Tell application "System Events" to display dialog "Enter password for Teleport user %s" default answer "" with hidden answer' -e 'text returned of result' 2>/dev/null)
				OTP=$(osascript -e 'Tell application "System Events" to display dialog "Enter your OTP token" default answer ""' -e 'text returned of result' 2>/dev/null)
				expect -c "
					spawn %s login --format=openssh
					expect \"Enter password for Teleport user\"
					send \"$PWD\r\"
					expect \"Enter your OTP token:\"
					send \"$OTP\r\"
					expect \"You are now logged into\"
					send \"logged in\"; exit 1
					expect \"invalid username, password or second factor\"
					send \"login failed\"; exit 1"`, teleport.User, baseCMD)
			result, err := exec.Command("bash", "-c", checkerCMD).CombinedOutput()
			if err == nil {
				if bytes.Contains(result, []byte(teleport.Cluster+" online")) {
					cmd = "Logout"
					status = "On"

					shellCMD = baseCMD + " logout"
				}
			}
			item := gofred.NewItem(name, cmd+" "+name, noAutocomplete).AddIcon(status+".png", "").
				AddVariables(gofred.NewVariable("name", name), gofred.NewVariable("cmd", cmd), gofred.NewVariable("checker", checkerCMD)).
				AddCommandKeyAction("Modify config", "modify", true).
				AddCommandKeyVariables(gofred.NewVariable("name", name), gofred.NewVariable("cmd", "modify")).
				AddOptionKeyAction("Remove config", "remove", true).
				AddOptionKeyVariables(gofred.NewVariable("name", name), gofred.NewVariable("cmd", "remove")).Executable(shellCMD)
			items = append(items, item)
		}

		items = append(items, gofred.NewItem("Add new config", noSubtitle, "create ").AddIcon("plus.png", ""))
	} else {
		response.AddVariable("filename", flag.Arg(1))
		items = append(items, gofred.NewItem("Add new config", fmt.Sprintf("write name ... \"%s\"", flag.Arg(1)), noAutocomplete).
			AddIcon("plus.png", "").Executable("new").AddVariables(gofred.NewVariable("cmd", "new")))
	}
	response.AddItems(items...)
	fmt.Println(response)
}

func (c Config) isValid() bool {
	if len(c.Proxy) == 0 {
		return false
	}
	return true
}
func (c Config) baseCMD() string {
	cmd := fmt.Sprintf("/usr/local/bin/tsh --proxy=%s", c.Proxy)
	if len(c.Cluster) > 0 {
		cmd += fmt.Sprintf(" --cluster=%s", c.Cluster)
	}
	if len(c.User) > 0 {
		cmd += fmt.Sprintf(" --user=%s", c.User)
	}
	if len(c.Auth) > 0 {
		cmd += fmt.Sprintf(" --auth=%s", c.Auth)
	}

	return cmd
}
