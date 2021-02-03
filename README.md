# Amazon EC2 System Monitor for macOS


## Overview
Amazon EC2 System Monitor for macOS is a small agent that runs on every [mac1.metal](https://aws.amazon.com/ec2/instance-types/mac/)
instance to provide on-instance metrics in CloudWatch. Currently the primary use case for this agent is to send CPU utilization
metrics. This uses a serial connection attached via the [AWS Nitro System](https://aws.amazon.com/ec2/nitro/) 
and is forwarded to CloudWatch for the instance automatically. 

## Usage
The agent is installed and enabled by default for all AMIs vended by AWS. It logs to 
`/var/log/amazon/ec2/system-monitoring.log` and can be updated via [Homebrew](https://github.com/aws/homebrew-aws). 

### Managing the monitor with `setup-ec2monitoring`
The package includes a shell script for enabling, disabling, and listing the current status of the agent 
according to `launchd`. 

### Viewing the agent status
To view the status of the agent:
```bash
sudo setup-ec2monitoring list
```

### Enabling the agent
To enable/install ec2-macos-system-monitor:
```bash
sudo setup-ec2monitoring enable
```
This must be run if updating to a new version to ensure it is scheduled to run again. 

### Disabling the agent
To disable ec2-macos-system-monitor:
```bash
sudo setup-ec2monitoring disable
```

## Design
The Amazon EC2 System Monitor for macOS uses multiple goroutines to manage two primary mechanisms:
1. The serial relay takes data from a UNIX domain socket and writes the data in a payload via a basic wire protocol.
2. Runs a ticker that reads CPU utilization and sends the CPU usage percentage to the UNIX domain socket.

This design allows for multiple different processes to write to the serial device while allowing one process to
always have the device open for writing. 

### Wire protocol
The wire protocol's primary purpose is to ensure the payload is complete by wrapping the payload in a checksum.
There is a tag which is used as a namespace to ensure the reader knows what type of data is being written. The data
itself which, in this case is typically the CPU utilization as a percentage of total usage the second field. Finally, a 
boolean is set specifying if the data should be compressed before sending. A checksum is then computed on this payload 
and included along with the payload. This payload with checksum allows the receiver to ensure that all the data was 
correctly received as well as inform if the data should be decompressed before parsing.

## Security

See [CONTRIBUTING](CONTRIBUTING.md#security-issue-notifications) for more information.

## License

This project is licensed under the [Apache License, version 2.0](https://www.apache.org/licenses/LICENSE-2.0).

