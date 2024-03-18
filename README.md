# aws-all

A golang tool to issue a command against multiple aws profiles in concurrency while combining results in JSON format.

Works with AWS CLI commands that return JSON formatted data.

Does not support AWS CLI `--query` argument yet. Probably not others also.

Specify an "AWS_ALL" environemnt variable that is a string or regex for the profile names you want to include in commands.

Example:
```
go build -o aws-all main.go

AWS_ALL="^per" ./aws-all ec2 describe-instances | jq -r '.[] | .Profile as $prof | .Results.Reservations[].Instances[] | [$prof, .InstanceId, .State.Name] | @csv'
"personal","i-0751a24bce64419a1","running"
"personal2","i-0751a24bce64419a1","running"
```