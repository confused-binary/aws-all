#!/usr/bin/python3

import asyncio
import subprocess
import sys
import os
import json
import csv


def print_help():
    print('## AWS-ALL Script: Run commands against multiple AWS accounts')
    print(' - Must set "ENG_HOME" environment variable so the script knows where to find the scope.csv file.')
    print(' - scope.csv should be a CSV text file with "profile,account" for each line.')


eng_home = os.environ.get('ENG_HOME')
if not eng_home:
    print_help()
    sys.exit(f"ENG_HOME environment variable is not set")
SCOPE_FILE = f"{eng_home}/scope.csv"
SCOPE = {}
if not os.path.exists(SCOPE_FILE):
    print_help()
    sys.exit(f"Scope file not found at {SCOPE_FILE}\nCreate with prof,acct csv-format.")

VALID_REGIONS = [
    "ap-northeast-1",
    "ap-northeast-2",
    "ap-northeast-3",
    "ap-south-1",
    "ap-southeast-1",
    "ap-southeast-2",
    "ca-central-1",
    "eu-central-1",
    "eu-north-1",
    "eu-west-1",
    "eu-west-2",
    "eu-west-3",
    "sa-east-1",
    "us-east-1",
    "us-east-2",
    "us-west-1",
    "us-west-2"
]

# Get valid local profiles
def validate_profiles():
    with open(f"{SCOPE_FILE}", newline="\n") as csvfile:
        reader = csv.reader(csvfile)
        for row in reader:
            profile = row[0]
            acct = row[1]
            SCOPE[profile] = acct

    invalid_profiles = []
    valid_profiles = []
    command = f"aws configure list-profiles".split(' ')
    result = subprocess.run(command, capture_output=True)
    stdout_output = result.stdout.decode("utf-8").split('\n')
    profiles = [a for a in stdout_output if a != ""]
    for prof in SCOPE.keys():
        if prof not in profiles:
            invalid_profiles.append(prof)
        else:
            valid_profiles.append(prof)
    if invalid_profiles:
        print_help()
        sys.exit(f"The following profiles in {SCOPE_FILE} aren't found as an AWS profile - {invalid_profiles}")
    return valid_profiles


def get_acct_id(profile):
    command = f"aws --profile {profile} --output json sts get-caller-identity".split(' ')
    result = subprocess.run(command, capture_output=True)
    stdout_output = result.stdout.decode("utf-8").split('\n')
    json_data = json.loads(''.join(stdout_output))
    acct_id = json_data.get('Account')
    with open(f"{SCOPE_FILE}", "a") as fp:
        fp.write(f"{profile},{acct_id}\n")
    fp.close()
    return acct_id


async def enrich_data(profile, prof_accts, command):
    check_commands = [a for a in command]
    regions = []
    count = 0
    for part in check_commands:
        if part == "--region":
            regions.append(check_commands[count+1])
            check_commands.pop(count)
            check_commands.pop(count)
        count = count + 1
    if regions and ',' in regions[0]:
        regions = regions[0].split(',')
    if regions and regions[0].lower() == 'all':
        region_command = f"aws --profile {profile} --region us-east-1 ec2 describe-regions".split(' ')
        result = subprocess.run(region_command, capture_output=True)
        result = result.stdout.decode("utf-8")
        json_data = json.loads(result).get('Regions')
        regions = [a.get('RegionName') for a in json_data]
    elif not regions:
        region_command = f"aws --profile {profile} configure get region".split(' ')
        result = subprocess.run(region_command, capture_output=True)
        regions = [a for a in result.stdout.decode("utf-8").split('\n') if a != '']
    elif 'default' in regions:
        region_command = f"aws --profile {profile} configure get region".split(' ')
        result = subprocess.run(region_command, capture_output=True)
        result = result.stdout.decode("utf-8").strip('\n')
        regions = [a if a != "default" else result for a in regions]
    false_regions = [a for a in regions if a not in VALID_REGIONS]
    if false_regions:
        print_help()
        sys.exit(f"Invalid region(s) found: {','.join(false_regions)}")
    if not regions:
        print_help()
        sys.exit(f"No valid region found")
    if profile not in prof_accts:
        acct_id = get_acct_id(profile)
    else:
        acct_id = prof_accts.get(profile)
    regions = list(dict.fromkeys(regions))
    return {"Profile": profile,
            "Account": acct_id,
            "Regions": regions,
            "Command": check_commands}


async def run_command(profile, account, regions, command):
    region_data = []
    for region in regions:
        command_prefix = f"aws --profile {profile} --region {region}"
        running_command = f"{command_prefix} {' '.join(command)}"
        running_command = running_command.split(' ')    
        no_json_commands = [
            "s3"
        ]
        no_json_cmd = [a for a in no_json_commands if command[0] == a]
        if no_json_cmd:
            # jsut print since json output is not possible
            result = subprocess.run(running_command, capture_output = True)
            stdout = result.stdout.decode("utf-8")
            stderr = result.stderr.decode("utf-8")
            data = {"no-json-result": stdout}
            if stderr:
                print(running_command)
                sys.exit(stderr)
        else:
            # capture as json and print as such
            running_command = running_command + ["--output", "json"]
            running_command = [a.strip("'") for a in running_command]
            result = subprocess.run(running_command, capture_output = True)
            stdout = result.stdout.decode("utf-8").split("\n")
            stderr = result.stderr.decode("utf-8")
            if stderr:
                print(running_command)
                sys.exit(stderr)
            if "--query" in running_command:
                if len(stdout) <= 2:
                    stdout = '[]'
                title_guess = ''.join([a.capitalize() for a in command[1].split('-')[1:]])
                data = {title_guess: sum(json.loads(''.join(stdout)), [])}
            else:
                data = json.loads(''.join(stdout))

        data['Profile'] = profile
        data['Account'] = account
        data['Region'] = region
        region_data.append(data)
    return region_data


async def main():
    valid_profiles = validate_profiles()
    prof_accts = {}
    if os.path.exists(f"{SCOPE_FILE}"):
        with open(f"{SCOPE_FILE}", "r") as fp:
            for line in fp:
                prof = line.split(',')[0]
                acct = line.split(',')[1].strip('\n')
                prof_accts[prof] = acct
    command = sys.argv[1:]
    for item in command:
        if item.lower() == "-h" or item.lower() == "--help":
            print_help()
            sys.exit()   
    if not command:
        print_help()
        sys.exit(f"I need to know what aws command to run...")
    
    tasks = []
    for profile in valid_profiles:
        tasks.append(asyncio.ensure_future(enrich_data(profile, prof_accts, command)))
    data = await asyncio.gather(*tasks)
    
    tasks = []
    for item in data:
        profile = item.get('Profile')
        account = item.get('Account')
        regions = item.get('Regions')
        command = item.get('Command')
        tasks.append(asyncio.ensure_future(run_command(profile, account, regions, command)))
    results = await asyncio.gather(*tasks)
    results = sum(results, [])
    if 'no-json-result' in sum([list(a.keys()) for a in results], []):
        for result in results:
            print(f"Profile: {result.get('Profile')}")
            print(f"Account: {result.get('Account')}")
            print(f"Region: {result.get('Region')}")
            print(result.get('no-json-result'))
    else:
        formatted_json = json.dumps(results, indent=2)
        print(formatted_json)

if __name__ == "__main__":
    loop = asyncio.new_event_loop()
    asyncio.set_event_loop(loop)
    loop.run_until_complete(main())
    loop.close()
