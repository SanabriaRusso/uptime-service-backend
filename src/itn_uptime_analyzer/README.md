ITN Uptime Analyzer
===================

The ITN Uptime Analyzer is a program which browses uptime submissions
in an AWS S3 bucket and determines the total uptime of each block
producer. The reason for measuring nodes' uptime during the ITN is
that the Foundation pays Mina tokens as a reward for BPs for
participation (hence the "incentivised" test network).

The uptime protocol
-------------------

The protocol demands that each block producer partaking in the ITN
should submit a request to the uptime service as a proof of them
running a node.  A request should be submitted every 15 minutes and
the daemon contains the functionality which, given an uptime service
URL, sends these submissions automatically.

The uptime service stores the submissions verbatim in an AWS S3 bucket
for later analysis. Each submission is a JSON file containing node's
public key, IP address, GraphQL port, a signature and a timestamp.
Submitters are identified by their:
* public key
* IP address
* GraphQL port

**NOTE**: the program can be configured to ignore IP addresses and
ports, and take public keys as sole identifiers for each node.

Given some time bounds, the program looks up files submitted by each
identity and counts submissions. Any submission sent within less than
10 minutes from the previous counted one is ignored. Finally,
submissions sent be the identity within given time period are divided
by the number of submissions expected for the time period (one per 15
minutes). All the identities together with their percentage scores are
output to standard output in CSV format.

An example CSV:
```
Period start; 2023-10-17 12:00:00 +0000 UTC
Period end; 2023-10-18 00:00:00 +0000 UTC
Interval; 12h0m0sa
public key; uptime (%)
B62qidtaSSb15kGDWewE3mXavp72D36e3s6tY1tGyY7TYzgi1b4s2Ct; 100.00
B62qinbabrSwrDmz2FXDXY9K7wjC8WLuDSRu2e4ENKtA5CUtrNhYW9g; 100.00
B62qjJeSHaWVyLeVeDVkn8kqbTU9qsnKniHoXGJMN1ZiGaxeDBX47Su; 100.00
B62qkDizWLQD2j59kaJG5LdMJSUvooLQmoGAb8oWeZ5wqypwMdUo74H; 100.00
B62qkRhgKuqyR1RfFmDx3ckP4eRfzeab81wuU2TVbJddDcC7w9nWHMK; 100.00
B62qkpK35G1W1dn2136nNGtR9CM5suYD6EvTgC4NaNp9xyC9NuC1xEA; 100.00
B62qkxksE59PKtd7J6gGEfMdhBDSzR5iWBmoMkxwnRGNBHHp6NJ7veX; 100.00
B62qmQUxBSRBUAvisBCxMjsfekAP5Vasp7G4VgzH4dFUyEEQxjBpbTy; 100.00
B62qnATYbV6CN6PZ6micywtJ4bPMHXjWthL9oS19rE7pEhNoqBqYMT9; 100.00
B62qnND2TWcAo9rxDhUGEvMmKpqau7ubQjAneYSNfDiRZRJ2wPWxj6T; 100.00
B62qnRDGrKE1tD5pL8CCzQ66wGbA8XgJbD1s3A3Kg1BFSBwBmBePKFg; 100.00
B62qnd4SfzexmHnPjoRtPvksVHn1bicLRxBr1BPDSozUEUmxsQqU8iR; 100.00
B62qnkcuZVFSHhYMsxKAkjcuTi3eeok5R9pFqDDuoANFmbrefGSE4eS; 100.00
B62qnnUBr6YCeRMTjnpu3WrhaB88T5su6CKnKUurtuTjhKfVoRL6YnX; 100.00
```
IP addresses (if taken into account, are placed in a column between
public keys and uptime percentages).

Compilation
-----------

Assuming you are in the Nix shell described in the main directory
README, enter this directory and type:

    $ go build
    
Running
-------

The program can be configured either using a JSON file or environment
variables (but not both). Program does not accept any command line
arguments. If a configuration file is specified (see below),
environment variables are ignored.

AWS access credentials can be loaded from a file (define an
environment variable `AWS_CREDENTIALS_FILE`) or using aws-cli. Just
install the package, configure it so that it can access the S3 bucket
and this program will use it.

Environment configuration
-------------------------

If the `CONFIG_FILE` variable is undefined or empty, program will try
reading the configuration from its environment. The following
variables are mandatory – failing to define any one of them will
result in an error:
* `CONFIG_AWS_REGION` - AWS region in which to look for the S3 bucket.
* `CONFIG_AWS_ACCOUNT_ID` - AWS account identifier to log into.
* `CONFIG_NETWORK_NAME` – name of the network to browse.

The network name is a name of the AWS S3 bucket's subdirectory in
which to look for submissions.

Additionally the following optional variables may be defined:
* `CONFIG_IGNORE_IPS` – if not empty it tells the program to ignore
  submissions' IP addresses and ports (see above)
* `CONFIG_PERIOD_START`
* `CONFIG_PERIOD_END`
* `CONFIG_PERIOD_INTERVAL`

For the explanation on the execution period, see the relevant section
below.

JSON file configuration
-----------------------

In order to use a configuration file, define an environment variable
`CONFIG_FILE`. The variable should contain the path to the
configuration file. The configuration file can look like this:

```
{
  "aws": {
    "region": "us-west-2",
    "account_id": "673156464838"
  },
  "period": {
    "start": "2023-10-20T00:00:00Z",
    "end": "2023-10-20T12:00:00Z",
    "interval": 720
  },
  "network_name": "pre-itn-1",
  "ignore_ips": true
}
```
All the fields under `aws` key as well as `network_name` are mandatory.
Failing to specify them will result in an error.

The field `ignore_ips` is optional and defaults to `false`.

For explanation regarding `period`, see below.

Execution period
----------------

Execution period configuration consists of three options. If all 3 are
defined, they must match, that is the interval (expressed in minutes)
must be the exact time interval between start and end. If that's not
the case, the program will stop with and error.

If any 2 of them are defined, the third will be adjusted so that the
rule above holds.

The default values are as follows:
* interval - 12h or 720 min
* end - if it's before noon, midnight of the current day, otherwise
  the noon of the current day
* start – interval (12h by default) before the end

If only the start is defined, the end is assumed to be 12h later.  If
only end or interval is specified, the other two options take the
default values (see above).

**IMPORTANT**: because in production the script never needs to access
data from multiple days, that feature is not implemented. If the
execution period spans over multiple days, only data from the
**first** day will be analysed.
