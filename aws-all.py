#!/usr/bin/python3

import asyncio
import subprocess
import sys
import os
import json
import csv
import argparse
import traceback
import logging
import shutil

SCOPE = {}

# https://docs.aws.amazon.com/whitepapers/latest/aws-fault-isolation-boundaries/global-services.html
GLOBAL_SERVICES = {
    'iam': 'us-east-1', 
    'organizations': 'us-east-1',
    'account': 'us-east-1',
    'route53-recovery-cluster': 'us-west-2',
    'networkmanager': 'us-west-2',
    'route53': 'us-east-1',
    's3': 'us-east-1',
    's3api': 'us-east-1'
}


def list_str(values):
    '''
    Just split csv string to list
    '''
    return values.split(',')


class Account:
    def __init__(self, profile, id, regions, results=None, no_json=False) -> None:
        self.profile = profile
        self.id = id
        self.regions = regions
        self.results = results
        self.no_json = no_json

    def __str__(self):
        return f'{self.profile}-{self.id}'


class validate():
    def __init__(self) -> None:
        pass


    def aws_cli():
        '''
        Make sure aws cli is installed locally
        '''
        if not shutil.which('aws'):
            logging.error(f'The AWS CLI was not found. Install the CLI to use this tool.')
            sys.exit()


    def cli_args():
        '''
        Process CLI Arguments
        '''
        parser = argparse.ArgumentParser(prog='aws-all',
                                        description='runs one command against multiple AWS profiles',
                                        epilog='Hopefully this works as intended...')
        parser.add_argument('-p', '--profiles', '--profile', type=list_str,
                            help='CSV style list of profiles Ex. "example1,example2" (No Default)')
        parser.add_argument('-r', '--regions', '--region', type=list_str,
                            help='"all" for all regions or CSV style list of regions Ex. "us-east-1,eu-west-2" (Defaults to default region for profile)')
        parser.add_argument('-s', '--scope-file', help='Locatin of the scope file to read from (Defaults to profiles listed in AA_SCOPE ENV-VAR file)')
        parser.add_argument('-o', '--global-override', action='store_true', help='Override global service region adjustment to instead check all provided regions')
        parser.add_argument('-a', '--add-to-scope', action='store_true', help='Add profiles to the scope file if one is set')
        parser.add_argument('command', nargs='+', help='The AWS CLI command to run')
        global CLI_ARGS
        CLI_ARGS, unknown = parser.parse_known_args()

        if unknown:
            CLI_ARGS.command = CLI_ARGS.command + unknown

        if CLI_ARGS.profiles is not None and CLI_ARGS.scope_file is not None:
            logging.error(f'Can not pass both "profiles" and "scope"')
            sys.exit()

        if not CLI_ARGS.scope_file:
            CLI_ARGS.scope_file = os.getenv('AA_SCOPE')


    def profiles():
        '''
        Get list of locally cconfigured profiles and return those that are valid
        '''
        local_profiles = run_command.get_local_profiles()
        valid_profiles = [a for a in CLI_ARGS.profiles if a in local_profiles]
        invalid_profiles = [a for a in CLI_ARGS.profiles if a not in valid_profiles] 
        if invalid_profiles:
            logging.error(f'Invalid profiles specified: {",".join(invalid_profiles)}')
            sys.exit()
        
        return valid_profiles


    def regions(profile):
        '''
        Get list of locally cconfigured profiles and return those that are valid
        '''
        aws_regions = ['af-south-1',
                       'ap-east-1', 'ap-northeast-1', 'ap-northeast-2', 'ap-northeast-3',
                       'ap-south-1', 'ap-south-2', 'ap-southeast-1', 'ap-southeast-2', 'ap-southeast-3', 'ap-southeast-4',
                       'ca-central-1', 'ca-west-1',
                       'eu-central-1', 'eu-central-2', 'eu-north-1', 'eu-south-1', 'eu-south-2', 'eu-west-1', 'eu-west-2', 'eu-west-3',
                       'il-central-1',
                       'me-central-1', 'me-south-1',
                       'sa-east-1',
                       'us-east-1', 'us-east-2', 'us-gov-east-1', 'us-gov-west-1', 'us-west-1', 'us-west-2']

        if CLI_ARGS.regions is None:
            default_region = run_command.get_default_region(profile)
            if [a for a in default_region if a in aws_regions]:
                return default_region
            else:
                logging.error(f'Invalid or no region specified for profile.')
                sys.exit()
        
        if CLI_ARGS.command[0] in GLOBAL_SERVICES.keys() and not CLI_ARGS.global_override:
            # If command is for global service, just use that service's associated region
            valid_regions = [GLOBAL_SERVICES.get(CLI_ARGS.command[0])]
        elif CLI_ARGS.regions and 'all' in CLI_ARGS.regions:
            valid_regions = aws_regions
        else:
            valid_regions = [a for a in CLI_ARGS.regions if a in aws_regions]
            invalid_regions = [a for a in CLI_ARGS.regions if a not in aws_regions] 
            if invalid_regions:
                logging.error(f'Invalid regions specified: {",".join(invalid_regions)}')
                sys.exit()
        
        return valid_regions


    def scope_env():
        '''
        Get profile details from SCOPE ENV-VAR
        '''
        try:
            CLI_ARGS.scope_file = os.getenv('AA_SCOPE')
            if not CLI_ARGS.scope_file:
                logging.error(f'"AA_SCOPE" Environment Variable is not set.')
                sys.exit()
        except Exception:
            logging.error(f'Error checking scope Environment Variable')
            sys.exit()
        return run_command.read_scope_file()


    async def build_account_obj(profile):
        '''
        Build the Account objects
        '''
        valid_regions = validate.regions(profile[0])
        account = Account(profile=profile[0], id=profile[1], regions=valid_regions)
        return account


    async def accounts():
        '''
        Build list of valid accounts for scope
        '''
        if CLI_ARGS.profiles is not None:
            valid_profiles = validate.profiles()
            CLI_ARGS.scope = [( a, run_command.acct_id(a) ) for a in valid_profiles]
        elif CLI_ARGS.scope_file is not None:
            CLI_ARGS.scope = run_command.read_scope_file()
        else:
            CLI_ARGS.scope = validate.scope_env()
        
        global valid_scope
        tasks = []
        for profile in CLI_ARGS.scope:
            tasks.append(
                asyncio.ensure_future(
                    validate.build_account_obj(profile)))
        valid_scope = await asyncio.gather(*tasks)
        return valid_scope
            # valid_regions = validate.regions(profile)
            # account = Account(profile=profile[0], id=profile[1], regions=valid_regions)
            # valid_scope.append(account)


class run_command():
    def __init__(self) -> None:
        pass


    def acct_id(profile):
            '''
            Get and return account id for provided profile
            '''
            acct_id = None
            if 'scope' in CLI_ARGS:
                acct_id = ''.join(a[1] for a in CLI_ARGS.scope if a[0] == profile)
            elif CLI_ARGS.scope_file:
                acct_id = ''.join(a[1] for a in validate.scope_env() if a[0] == profile)
            if not acct_id:
                acct_id = run_command.get_acct_id_from_aws(profile)
            if CLI_ARGS.add_to_scope:
                run_command.write_scope_file(f'{profile},{acct_id}\n')
            return acct_id


    def get_local_profiles():
        '''
        Run AWS CLI command to get list of default profiles
        '''
        try:
            command = f'aws configure list-profiles'.split(' ')
            result = subprocess.run(command, capture_output=True)
            stdout_output = result.stdout.decode('utf-8').split('\n')
            local_profiles = list(filter(None, stdout_output))
        except Exception:
            logging.error(traceback.format_exc())
            sys.exit()
        return local_profiles


    def get_default_region(profile):
        '''
        Run AWS CLI command to get the configured default region for the provided profile
        '''
        try:
            command = f'aws --profile {profile} configure get region'.split(' ')
            result = subprocess.run(command, capture_output=True)
            stdout_output = result.stdout.decode('utf-8').split('\n')
            regions = list(filter(None, stdout_output))
        except Exception:
            logging.error(traceback.format_exc())
            sys.exit()
        return regions


    def get_acct_id_from_aws(profile):
        '''
        Run AWS CLI command to get Profile ACCT ID
        '''
        try:
            command = f'aws --profile {profile} --output json sts get-caller-identity'.split(' ')
            result = subprocess.run(command, capture_output=True)
            stdout_output = result.stdout.decode('utf-8').split('\n')
            json_data = json.loads(''.join(stdout_output))
        except Exception:
            logging.error(traceback.format_exc())
            sys.exit()
        return json_data.get('Account')


    def read_scope_file():
        '''
        Read scope data from provided scope_file location
        '''
        scope_data = []
        try:
            with open(f'{CLI_ARGS.scope_file}', newline='\n') as csvfile:
                reader = csv.reader(csvfile)
                csv_data = [row for row in reader if row]
                scope_data = [( row[0], row[1] ) for row in csv_data if row[0].lower() != 'profile']
        except FileNotFoundError:
            logging.error(f'No scope file was found at {CLI_ARGS.scope_file}')
            sys.exit()
        return scope_data


    def write_scope_file(entry):
        '''
        Add acct to scope file
        '''
        try:
            with open(f'{CLI_ARGS.scope_file}', 'a') as fp:
                fp.write(entry)
            fp.close()
        except Exception:
            logging.error(traceback.format_exc())
            sys.exit()


    async def async_command(valid_scope):
        '''
        Execute the provided command
        '''
        tasks = []
        for account in valid_scope:
            tasks.append(asyncio.ensure_future(run_command.async_task(account)))
        results = await asyncio.gather(*tasks)
        return results


    async def async_task(account):
        '''
        Setup asyncio task
        '''
        region_results = {}
        for region in account.regions:
            command = f'aws --profile {account.profile} --region {region} {" ".join(CLI_ARGS.command)} --output json'
            stdout = run_command.execute_command(command)

            try:
                if '--query' in CLI_ARGS.command:
                    if len(stdout) <= 2:
                        stdout = '[]'
                    title_guess = ''.join([a.capitalize() for a in CLI_ARGS.command[1].split('-')[1:]])
                    json_data = json.loads(''.join(stdout))
                    results = {title_guess: json_data}
                else:
                    results = json.loads(''.join(stdout))
            except ValueError:
                results = {'no-json-result': stdout}
                account.no_json = True
            region_results[region] = results
        
        account.results = region_results
        return account


    def execute_command(command):
        '''
        Finally execute the specified task and return results
        '''
        try:
            result = subprocess.run(command, shell=True, capture_output=True)
            stdout = result.stdout.decode('utf-8')
            stderr = result.stderr.decode('utf-8')
            if stderr:
                logging.error(command)
                raise Exception(stderr)
        except Exception:
            logging.error(traceback.format_exc())
            sys.exit()
        return stdout


    def print_results(account_results):
        '''
        Print results to stdout
        '''
        print_results = []
        for account in account_results:
            for region, results in account.results.items():
                data = {}
                data['Profile'] = account.profile
                data['Account'] = account.id
                data['Region'] = region
                print_results.append({**data, **results})
        if account.no_json:
            for result in print_results:
                print(f'Profile: {result.get("Profile")}')
                print(f'Account: {result.get("Account")}')
                print(f'Region: {result.get("Region")}')
                print(result.get('no-json-result'))
        else:
            formatted_json = json.dumps(print_results, indent=2)
            print(formatted_json)


async def main():
    # Make sure AWS CLI is installed
    validate.aws_cli()
    # Process CLI arguments, if any are provided
    validate.cli_args()

    # Validate details about provided accounts
    valid_scope = await validate.accounts()
    # Finally, run the AWS CLI command that was provided
    all_results = await run_command.async_command(valid_scope)
    
    # Print results to stdout
    run_command.print_results(all_results)


if __name__ == '__main__':
    loop = asyncio.new_event_loop()
    asyncio.set_event_loop(loop)
    loop.run_until_complete(main())
    loop.close()
