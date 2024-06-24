AWS-ALL

A python script that can be used to concurrently run awscli commands against multiple AWS accounts as defined by a specific scope.csv file.

## Setup
It should work as is. No need to install extra packages, but the AWS CLI will need to be installed.

AWS profiles will want to also already be setup in whichever way is convenient.

The profile names listed in scope.csv should match those returned with `aws configure list-profiles`.

The file can be renamed and added to a $PATH location for easier use.

```
sudo cp aws-all.py /usr/bin/aws-all
sudo chmod +x /usr/bin/aws-all
sudo chown root:root /usr/bin/aws-all
```

## Use

With the script set as executable and set in $PATH, it can be used like normal.

```
aws-all s3api list-buckets
```

Results will be in JSON format with some additional details added such as the profile name, account id, and region the command was completed in.

Any use of the `--region` argument is captured and used as input for running the provided command in those regions in the accounts in scope. Multiple regions can be specified in unspaced CSV format "us-east-1,us-east-2". There are also two shortcuts for regions argument.
- "defualt": Will use the default region found for the profile, as reportd by from `aws configure get region` command.
- "all": Will use all regions, as reported from `aws ec2 describe-regions` command.

Any erros reported by awscli should also be displayed normally along with a copy of the attempted command that caused the issue.
