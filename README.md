# rcma

rcma is the AWS-centric rcmd tool -- a simple golang binary that:
  - takes as input a comma-separated list of AWS tags to match on and a command 
  - includes ability to exclude based on certain tags
  - connects via ssh to all hosts that match the tag filter
  - runs the command on each host
  - returns the results

"Oh so you've recreated ansible?"  Kind of I guess.  This is a single binary + connection configuration.  I've used it more in emergencies or even to check states on hosts.

"But there's a thingie for ansible to do the same thing with AWS"  Great.

It was designed to be as simple as possible.  Zero work was done to support any AWS configs other than the default user files in ~/.aws.

```
rcma -error -privateip Name=*web*,Team=infra "ps aux | grep nginx | wc -l"
```

# rcmd -- original script

## Description

Command line tool to enact changes or makes queries across multiple linux hosts via ssh.  I wrote a python version of this originally to help manage legacy infrastructure at my day job.

## Features -- some not present

### Input
  - Pattern matching input against a list of host files
  - source can be a local file list, input list, or etcd url
  - specify a numeric range (coupled with the host match -- so host matches 'web' and we specify -range 100-123.  This assumes you use hostname convention like <location>-<host type>-<number>
  - list of files to exclude

### Output
  - dump to log file
  - json return value
  - web endpoint for very hardcoded values -- like nginx process count on all hosts dumps json blob of hostname + count?  maybe?  lots of security concerns -- worse than with the basic idea :)

