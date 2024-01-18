import os
import re
import csv
import sys
from collections import defaultdict

OUTPUT_SUBDIR = 'output'
FILENAME_RE = re.compile(
  'summary_(\d{4}-\d{2}-\d{2})T(\d{2}):\d{2}:\d{2}-\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.csv'
)

def process_csv_directory(directory):
    data = defaultdict(lambda: defaultdict(float))
    columns = set()
    output_data = {}

    csv_files = [file for file in os.listdir(directory) if file.endswith(".csv")]

    # This is number of half-days produced by script (needed for averaging availablity)
    base_for_avg = len(csv_files)

    for csv_file in csv_files:
        match = FILENAME_RE.match(csv_file)
        if not match:
            print(f"Skipping file: {csv_file}")
            continue
        period = match.group(1) + ("_AM" if match.group(2) == "00" else "_PM")
        columns.add(period)
        print(f"Processing file: {csv_file}")
        with open(os.path.join(directory, csv_file), 'r') as file:
            csv_reader = csv.reader(file, delimiter=';')

            # Skip header lines
            next(csv_reader)
            next(csv_reader)
            next(csv_reader)
            next(csv_reader)

            for row in csv_reader:
                key, value = row[0], float(row[1])
                data[key][period] = value

    columns = sorted(columns)

    # Save in dir/output as csv file
    partial_output_filename = os.path.join(directory, OUTPUT_SUBDIR, f"{directory}.partials.csv")
    with open(partial_output_filename, 'w') as partial_output_file:
        partial_output_file.write(f"BP key: {', '.join(columns)}\n")
        for key in data.keys():
            values = [data[key][column] for column in columns]
            print(f"{key}: {values}")
            partial_output_file.write(f"{key}: {', '.join(map(str, values))}\n")

    for key in data.keys():
        # Assumes correct input, no dups etc.
        submitted_values = len(data[key])
        avg_value = round(sum(data[key].values()) / max(base_for_avg, submitted_values), 2)
        output_data[key] = {'avg_value': avg_value, 'submitted': submitted_values}

    sorted_output_data = dict(sorted(output_data.items(), key=lambda item: item[1]['avg_value']))

    # Save in dir/output as csv file 
    output_filename = os.path.join(directory, OUTPUT_SUBDIR, f"{directory}.output.csv")
    with open(output_filename, 'w', newline='') as output_file:
        csv_writer = csv.writer(output_file)
        csv_writer.writerow(["BP key", "average %", "number of submissions"])
        for key, values in sorted_output_data.items():
            csv_writer.writerow([key, values['avg_value'], values['submitted']])

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: python script.py directory_name")
        sys.exit(1)

    directory = sys.argv[1]

    # Create subdir for results 
    output_directory = os.path.join(directory, OUTPUT_SUBDIR)
    os.makedirs(output_directory, exist_ok=True)
    
    process_csv_directory(directory)
