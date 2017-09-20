## rcmd

rcmd is the AWS-centric remote command tool based around parallel runs of ssh.  It is a simple golang binary that:
  - takes as input a comma-separated list of AWS tags to match on and a command 
  - includes ability to exclude based on certain tags
  - connects via ssh to all hosts that match the tag filter
  - runs the command on each host
  - returns the results

"Oh so you've recreated ansible?"  Kind of I guess.  This is a single binary + connection configuration.  I've used it more in emergencies or even to check states on hosts.

"But there's a thingie for ansible to do the same thing with AWS"  Great.

It was designed to be as simple as possible.  Zero work was done to support any AWS configs other than the default user files in ~/.aws.

## Releases

Binary Linux and OSX downloads found [here](https://github.com/tcotav/rcmd/releases).

## Examples

Match nodes tagged with `Name=*web*,Team=web` and run the command  `ps aux | grep nginx| wc -l`.  We pass it the `--quiet` flag which means keep it short in response.

```
rcmd -c ./config.json --quiet Name=*web*,Team=web "ps aux | grep nginx| wc -l"
```


Exclude tag example `-x Name=web`, match on `Team=web`.  `--privateip` tells rcmd to use the internal IP of the matched hosts.
 
```
rcmd -c ./config.json --privateip -x Name=web Team=web "date"
```

Same as the first example except we dump out json (in case you want to chain it with some other commands or automation).

```
rcmd -c ./config.json --json --privateip Name=*epoch*,Team=web "date"
```

The same as the first but only return responses that contain errors.
```
rcmd -error -privateip Name=*web*,Team=web "cat /home/ubuntu/file-doesn't-exist"
```

## Config

Sample in etc/config.json

```
{
  "user":"ubuntu",
  "keyfile":"/home/tcotav/.ssh/rcmd_test_key",
  "numworkers":10,
  "erroronly":false
}
```

	- user - ssh user connect to nodes with
	- keyfile - ssh key for the above user
	- numworkers - number of parallel ssh commands to run
	- erroronly - default of the print only errors flag

TODO - put some of the other switches here to set defaults.



