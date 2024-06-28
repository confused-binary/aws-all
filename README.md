AWS-ALL

A python script that can be used to concurrently run awscli commands against multiple AWS accounts and regions based on provided scope details.

## Details

During AWS penetration tests I'm often needing to run identical commands against multiple AWS accounts and multiple regions. So I created this tool to accomplish that a little faster while also providing output in a workable JSON format. I'll probably add to pip later after I update to add steps, but for now it can be copied to path and run from there as needed.

```
sudo cp aws-all.py /usr/local/bin/aws-all
sudo chmod +x /usr/local/bin/aws-all
sudo chown root:root /usr/local/bin/aws-all
```

## Prerequisites

1. Setup AWS CLI. This script is really a wrapper that just calls the CLI tool and processes it's stdout.
2. Setup AWS profiles. See [here](https://docs.aws.amazon.com/cli/v1/userguide/cli-chap-configure.html)
3. Optionally, create a scope file in CSV format that contains at least the name of each profile on each line. They should match the results from `aws configure list-profiles`
4. Optionally, set the Environment Variable "AA_SCOPE" to let aws-all know where to find the scope file. The scope file should list one profile name per line.

## Use
With the script set as executable and set in $PATH, it can be used like normal.

```
$ aws-all s3api list-buckets
```

### Arguments
- **profile**, **profiles**: Optionally specify the desired profile(s) on run. Will override using the profiles found in the scope-file
- **region**, **regions**: Optionally specify the desired region(s) on run. Will override using the default region as reported by `aws --profile {profile} configure get region`. Can specify "all" to run against all available AWS regions.
- **scope_file**: Optionally specify the scope file to use on run. Will override the "AA_SCOPE" Environment Variable.
- **global-override**: Global services, such as IAM, are flagged so that they are run against just one region even when multiple regions are specified. This boolean flag will override that, running the command against all regions.
- **add-to-scope**: This boolean flag will have the script add profiles specified with the `--profile` flag to the scope_file if they are missing from it.

Results are printed out in JSON format with some additional details added such as the profile name, account id, and region that the command was executed in.
