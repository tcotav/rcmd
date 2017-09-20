package main

import (
	"fmt"
//	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"net/url"
	"strings"
	"log"
	"flag"
)

const USAGE=`USAGE:
  Expected one or more key-value pair input: k1=v1 k2=v2
	Examples:
	  - rcmd Name=*web*    # can use limited regex
	  - rcmd Environment=Prod Name=*db*
`
const SPLIT="="


/* given a list of k=v pairs, generate list of servers that match those pairs */

type Hostdef struct {
	InstanceId string
	Name string
	PrivateIpAddress string
	InstanceType string
	PublicIpAddress string
}

func main() {
	//privateip := flag.Bool("privateip", false, "bool - use privateip.  defaults to public")
	flag.Parse()

	nArgs := flag.NArg()

	if nArgs == 0 {
		log.Print(USAGE)
		return
	}

	// create an array of ec2filters
	var filterArray []*ec2.Filter

	for _, kvpair := range flag.Args() {
		kvset := strings.Split(kvpair, SPLIT)
		if len(kvset)  != 2  {
			log.Print(USAGE)
			return
		}

		// values expects an []string
		var valList []*string
		valList = append(valList, &kvset[1])
		tagName := fmt.Sprintf("tag:%s",kvset[0])
		filterArray = append(filterArray, &ec2.Filter{Name: &tagName , Values: valList })
	}


	ec2svc := ec2.New(session.New())
	params := &ec2.DescribeInstancesInput{
		Filters: filterArray,
	}
	resp, err := ec2svc.DescribeInstances(params)
	if err != nil {
		fmt.Println("there was an error listing instances in", err.Error())
		log.Fatal(err.Error())
	}

	// Loop through the instances. They don't always have a name-tag so set it
	// to None if we can't find anything.
	for idx, _ := range resp.Reservations {
		for _, inst := range resp.Reservations[idx].Instances {

			// We need to see if the Name is one of the tags. It's not always
			// present and not required in Ec2.
			name := "None"
			for _, keys := range inst.Tags {
				if *keys.Key == "Name" {
					name = url.QueryEscape(*keys.Value)
				}
			}

			awshost := []*string{
				inst.InstanceId,
				&name,
				inst.PrivateIpAddress,
				inst.InstanceType,
				inst.PublicIpAddress,
			}

			// Convert any nil value to a printable string in case it doesn't
			// doesn't exist, which is the case with certain values
			output_vals := []string{}
			for _, val := range awshost {
				if val != nil {
					output_vals = append(output_vals, *val)
				} else {
					output_vals = append(output_vals, "None")
				}
			}
			// The values that we care about, in the order we want to print them
			fmt.Println(strings.Join(output_vals, " "))
		}
	}
}