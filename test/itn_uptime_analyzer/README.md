# ITN participation analysis script

## Why?

We are analyzing submissions to ITN's Uptime Service, to make sure ecosystem partners show up for ITN

## Flow

- Download submissions' summaries from s3 bucket
- Summaries are 1/2 day files
- Aggregate half day files into one report
- Upload report to google shared drive

## Aggregate script

1. Create directory and copy only correct summaries only. These have timestamp with T
    ```sh
    cp ~/downloads/FromS3/*T*.csv new-dir
    ```
2. Run aggregate script
    ```sh
    python3 ./aggregate.py new-dir
    ```
3. Inspect output and upload to shared location
    ```sh
    ls new-dir/output 
    new-dir.output.csv   new-dir.partials.csv
    ```
    
## Output

`new-dir.output.csv` file holds aggregated results from the beginning (expected)
`new-dir.partials.csv` holds half day breakdowns per BP, which would be helpful for deeper inspection 
Usually it is advisable to call `new-dir` a date which covers time aggregated 

