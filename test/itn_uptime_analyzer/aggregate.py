import os
import csv
import sys
from collections import defaultdict

OUTPUT_SUBDIR = 'output'

def process_csv_directory(directory):
    data = defaultdict(list)
    output_data = {}

    csv_files = [file for file in os.listdir(directory) if file.endswith(".csv")]

    # This is number of half-days produced by script (needed for averaging availablity)
    base_for_avg = len(csv_files)

    for csv_file in csv_files:
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
                data[key].append(value)

    # Save in dir/output as csv file
    partial_output_filename = os.path.join(directory, OUTPUT_SUBDIR, f"{directory}.partials.csv")
    with open(partial_output_filename, 'w') as partial_output_file:
        for key, values in data.items():
            print(f"{key}: {values}")
            partial_output_file.write(f"{key}: {values}\n")

    for key, values in data.items():
        # XXX still not accurate. It will undervalue duplicated entries per day
        submited_values = len(values)
        avg_value = round(sum(values) / max(base_for_avg, submited_values), 2)
        output_data[key] = {'avg_value': avg_value, 'submitted': len(values)}

    
    sorted_output_data = dict(sorted(output_data.items(), key=lambda item: item[1]['avg_value']))

    # Save in dir/output as csv file 
    output_filename = os.path.join(directory, OUTPUT_SUBDIR, f"{directory}.output.csv")
    with open(output_filename, 'w', newline='') as output_file:
        csv_writer = csv.writer(output_file)
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
