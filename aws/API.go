package aws

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	rcmd "github.com/tcotav/rcmd/client"
)

const SPLIT = "="

type Hostdef struct {
	InstanceId       string
	Name             string
	PrivateIpAddress string
	InstanceType     string
	PublicIpAddress  string
	Tags             []string
	Result           rcmd.HostCmdReturn
}

func GetHostList(inData string, excludeData string) ([]Hostdef, error) {

	// create an array of ec2filters
	var filterArray []*ec2.Filter

	for _, kvpair := range strings.Split(inData, ",") {
		kvset := strings.Split(kvpair, SPLIT)
		if len(kvset) != 2 {
			return nil, fmt.Errorf("invalid tag input: %s", kvpair)
		}

		// values expects an []string
		var valList []*string
		valList = append(valList, &kvset[1])
		tagName := fmt.Sprintf("tag:%s", kvset[0])
		filterArray = append(filterArray, &ec2.Filter{Name: &tagName, Values: valList})
	}

	ec2svc := ec2.New(session.New())
	params := &ec2.DescribeInstancesInput{
		Filters: filterArray,
	}
	resp, err := ec2svc.DescribeInstances(params)
	if err != nil {
		return nil, err
	}

	var excludeList map[string]string

	if excludeData != "" {
		excludeList = make(map[string]string)
		for _, kvpair := range strings.Split(excludeData, ",") {
			kvset := strings.Split(kvpair, SPLIT)
			// TODO -- what about matching any tag k regardless of value
			if len(kvset) != 2 {
				return nil, fmt.Errorf("invalid tag input: %s", kvpair)
			}
			excludeList[kvset[0]] = kvset[1]
		}
	}

	var retHosts []Hostdef

	// Loop through the instances. They don't always have a name-tag so set it
	// to None if we can't find anything.

	fSkipHost := false
	for idx, _ := range resp.Reservations {
		for _, inst := range resp.Reservations[idx].Instances {
			fSkipHost = false
			// We need to see if the Name is one of the tags. It's not always
			// present and not required in Ec2.
			name := "None"

			var tagList []string

			for _, keys := range inst.Tags {
				if *keys.Key == "Name" {
					name = url.QueryEscape(*keys.Value)
				}
				if _, ok := excludeList[*keys.Key]; ok { //match on key
					// now match regex the value
					excludeVal, _ := excludeList[*keys.Key]
					match, _ := regexp.MatchString(excludeVal, *keys.Value)
					if match {
						fSkipHost = true
						break // exclude this host because we matched k=v
					}
				}
				tagList = append(tagList, fmt.Sprintf("%s=%s", *keys.Key, *keys.Value))
			}

			if fSkipHost {
				continue
			}
			// confirm that
			publicIpAddress := ""

			if inst.PublicIpAddress != nil {
				publicIpAddress = *inst.PublicIpAddress
			}

			awshost := Hostdef{
				InstanceId:       *inst.InstanceId,
				Name:             name,
				PrivateIpAddress: *inst.PrivateIpAddress,
				InstanceType:     *inst.InstanceType,
				PublicIpAddress:  publicIpAddress,
				Tags:             tagList,
			}

			retHosts = append(retHosts, awshost)
		}
	}

	return retHosts, nil

}
