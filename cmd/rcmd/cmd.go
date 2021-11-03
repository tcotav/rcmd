package main

import (
	//"encoding/json"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"github.com/tcotav/rcmd/aws"
	rcmd "github.com/tcotav/rcmd/client"
)

const USAGE = `
	rcmd <flags> <aws tag match -- format k=v,k1=v1> "command here, ex: ls -la"
	Examples:
		rcmd -privateip -error Name=*web*,Team=infra \"ps aux | grep nginx | wc -l"
    rcmd -c ./config.json --quiet --privateip Name=*mywebsite*,Team=ops "date"
    rcmd -c ./config.json --quiet --privateip -x Name=*mywebsite* Team=ops "date"
    rcmd -c ./config.json --json --privateip -x Name=*mywebsite* Team=ops "date"

	Here's an example of rolling updates -- need both -r and -s.
	  --rolling/-r is the size of one set of hosts we're acting on
		--sleep/-s is the time to wait in between batches
	
		rcmd -c ./config.json --json --rolling 2 --sleep 2 --privateip Name=store-site-prod-host "ls -la | wc -l"
		rcmd -c ./config.json --json -r 2 -s 2 --privateip Name=store-site-prod-host "ls -la | wc -l"

	Help at:
		rcmd -h
`

var useConfig = flag.String("config", "", "string -- full path to config.json")
var excludeTags = flag.String("excludetags", "", "string of tags to exclude, expect k=v,k1=v1")

var rollSleepTime = flag.Int("sleep", 0, "time to sleep betweens sets during rolling update")
var rolling = flag.Int("rolling", 0, "work set size in rolling update")

func init() {
	// example with short version for long flag
	flag.StringVar(useConfig, "c", "", "string -- full path to config.json")
	flag.StringVar(excludeTags, "x", "", "string of tags to exclude, expect k=v,k1=v1")

	flag.IntVar(rollSleepTime, "s", 0, "time to sleep betweens sets during rolling update")
	flag.IntVar(rolling, "r", 0, "work set size in rolling update")
}

type JsonResults struct {
	Failure int
	Success int
	Total   int
	Cmd     string
	Results map[string]aws.Hostdef
}

func main() {
	privateip := flag.Bool("privateip", false, "bool - use privateip.  defaults to public")
	//prettyPrint := flag.Bool("pretty", false, "bool - pretty print")
	errorOnly := flag.Bool("error", false, "bool - print only errors")
	quiet := flag.Bool("quiet", false, "bool - print more node information")
	jsonFlag := flag.Bool("json", false, "bool - json output")

	flag.Parse()
	// we want to set up the series of keys that we'lg.Parse()
	argList := flag.Args()
	if flag.NArg() < 2 {
		fmt.Print(USAGE)
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *useConfig != "" {
		dirName, fileName := filepath.Split(*useConfig)
		fileParts := strings.Split(fileName, ".")
		viper.SetConfigName(fileParts[0])
		viper.AddConfigPath(dirName)
	} else {
		// then contain the next set
		viper.SetConfigName("config")
		viper.AddConfigPath("$HOME/.rcmd")
		viper.AddConfigPath("/etc/rcmd")
	}
	err := viper.ReadInConfig()

	if err != nil {
		fmt.Println("No configuration file loaded - aborting", err.Error())
		os.Exit(1)
	}

	viper.SetDefault("numworkers", 10)
	viper.SetDefault("sleeptime", 0)

	// get ssh key location from config file
	// ssh as user
	var ssh_user string
	if viper.IsSet("user") {
		ssh_user = viper.Get("user").(string)
	}
	if ssh_user == "" {
		// use the environment value for USER
		ssh_user = os.Getenv("USER")
	}
	ssh_keyfile := viper.Get("keyfile").(string)

	// Get list of hosts that match tags and excludes from AWS
	// returns a list of Hostdefs
	awsHostInfoList, err := aws.GetHostList(argList[0], *excludeTags)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if len(awsHostInfoList) == 0 {
		fmt.Println("No matching hosts found.")
		os.Exit(0)
	}

	var hostList []string
	// create this to be able to easily merge back later
	hostMap := make(map[string]aws.Hostdef)
	if *privateip {
		for _, h := range awsHostInfoList {
			hostList = append(hostList, h.PrivateIpAddress)
			hostMap[h.PrivateIpAddress] = h
		}
	} else { //default to using the public ip
		for _, h := range awsHostInfoList {
			hostList = append(hostList, h.PublicIpAddress)
			hostMap[h.PublicIpAddress] = h
		}
	}

	// change to config based
	numWorkers := viper.GetInt("numworkers")

	// we want to do a rolling update
	if *rollSleepTime != 0 && *rolling != 0 {
		numWorkers = *rolling
	}

	/*
		TODO -- a json option
	*/
	ret := rcmd.ProcessList(hostList, numWorkers, ssh_user, ssh_keyfile, argList[1], *errorOnly, *rollSleepTime)
	if *jsonFlag { // json report output
		for _, h := range ret.HostList {
			// remove the :22 from the string
			hhost := strings.Split(h.Host, ":")
			hostIp := hhost[0]
			awsHostInfo := hostMap[hostIp]
			awsHostInfo.Result = h
			hostMap[hostIp] = awsHostInfo

		}
		jsonRes := JsonResults{Failure: ret.Summary["failures"], Success: ret.Summary["success"], Total: ret.Summary["total"], Cmd: argList[1], Results: hostMap}
		b, _ := json.MarshalIndent(jsonRes, "", "  ")
		fmt.Println(string(b))

	} else {
		fmt.Println("------------------------------")
		fmt.Println("Summary:")
		fmt.Printf("Failure: %d, Success: %d, Total: %d\n", ret.Summary["failures"], ret.Summary["success"], ret.Summary["total"])
		for _, h := range ret.HostList {
			// remove the :22 from the string
			hhost := strings.Split(h.Host, ":")
			hostIp := hhost[0]
			fmt.Println("------------------------------")
			awsHostInfo := hostMap[hostIp]
			fmt.Printf("Name: %s\n", awsHostInfo.Name)
			fmt.Printf("IP: %s\n", hostIp)
			if !*quiet {
				fmt.Printf("InstanceType: %s\n", awsHostInfo.InstanceType)
				fmt.Printf("Tags: %s\n", strings.Join(awsHostInfo.Tags, ","))
			}
			if h.Stderr != "" {
				fmt.Println("  + ", h.Stderr)
			} else {
				for _, ll := range strings.Split(h.Stdout, "\n") {
					if ll != "" {
						fmt.Println("  - ", ll)
					}
				}
			}
		}
	}
}
