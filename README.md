# Agent Scheduler

Agent Scheduler is a Go-based application designed to calculate and schedule agent requirements for customer support operations across multiple timezones. It takes call volume data as input and generates a detailed hourly schedule of required agents, handling complex scenarios like partial hours, capacity constraints, and priority-based allocation.

## Features

-   **Multi-Timezone Support**: Handles input times in various timezones (e.g., "America/New_York", "Asia/Tokyo") and normalizes them for scheduling.
-   **Precise Scheduling Logic**:
    -   **Hourly Boundaries**: Schedules agents at clean hourly boundaries (e.g., 9:00, 10:00) regardless of exact start/end times.
    -   **Proportional Allocation**: Correctly handles partial hours (e.g., a shift starting at 9:30 AM) by allocating agents proportional to the time worked in that hour.
-   **Capacity Management**:
    -   **Capacity Constraints**: Supports a maximum global capacity per hour.
    -   **Priority-Based Allocation**: When demand exceeds capacity, agents are allocated to higher-priority customers first.
    -   **Unmet Demand Tracking**: Detailed reporting of unmet demand and impacted clients when capacity is limited.
-   **Utilization Adjustments**: Supports a utilization multiplier (0-1) to adjust agent requirements based on expected efficiency.
-   **Multiple Output Formats**: Generates schedules in Text, JSON, or CSV formats.

## Scheduling Logic

The scheduler uses a sophisticated algorithm to determine agent requirements:

1.  **Duration Calculation**: Calculates the total duration of the call window.
2.  **Call Distribution**: Distributes total calls evenly across the duration to determine calls per hour.
3.  **Hourly Iteration**: Iterates through the time window at hourly boundaries (e.g., 9:00, 10:00).
4.  **Proportional Calculation**: For each hour slot, it calculates the fraction of the hour actually used (e.g., 9:30-10:00 is 0.5 hours).
5.  **Agent Requirement**:
    -   `Calls This Hour = Calls Per Hour * Fraction of Hour Used`
    -   `Agents = Ceil(Calls This Hour * Average Duration / 3600)`
    -   `Adjusted Agents = Ceil(Agents / Utilization)`

## Usage

### Using Make (Recommended)

You can use the provided `Makefile` to build, run, and test the application easily.

-   **Build**: `make build`
-   **Run**: `make run` (Runs with default example data)
    -   To run with a specific input file: `make run INPUT=testdata/my_file.csv`
-   **Test**: `make test`
-   **Clean**: `make clean`

### Manual Build & Run

Build the application:

```bash
go build -o agent-scheduler main.go
```

Run the scheduler:

```bash
./agent-scheduler -input <input_file.csv> [flags]
```

### Flags

-   `-input`: Path to the input CSV file (Required).
-   `-format`: Output format: `text`, `json`, or `csv` (Default: `text`).
-   `-utilization`: Utilization multiplier between 0 and 1 (Default: `1.0`).
-   `-capacity`: Maximum agent capacity per hour (0 = unlimited).

### Example

```bash
./agent-scheduler -input testdata/data.csv -format csv -capacity 50
```

## Input Format

The input CSV file should have the following columns (headers are optional if using the parser that skips them, but standard format is recommended):

```csv
CustomerName, AverageCallDurationSeconds, StartTime, EndTime, NumberOfCalls, Priority, Timezone
```

-   **CustomerName**: Name of the client/project.
-   **AverageCallDurationSeconds**: Average handle time in seconds.
-   **StartTime/EndTime**: Time strings (e.g., "9:00AM", "15:30").
-   **NumberOfCalls**: Total calls expected in the window.
-   **Priority**: Integer priority (1 is highest).
-   **Timezone**: IANA timezone identifier (e.g., "America/New_York").

## Output Formats

### CSV
Produces a clean, one-row-per-hour format suitable for spreadsheet analysis:
```csv
Hour,Total Agents,Locations,Customer Details,Capacity Warning,Total Demand,Allocated,Unmet,Impacted Clients
09:00,30,Asia/Tokyo,"Tokyo Support(Asia/Tokyo,agents=30)",Yes,1491,30,1461,"Tokyo Support(priority=1...)"
```

### Text
Human-readable hourly breakdown:
```text
09:00 : total=30 ; [Asia/Tokyo: total=30, Tokyo Support=30]
  ⚠️  CAPACITY WARNING: Demand=1491, Allocated=30, Unmet=1461
  Impacted clients:
    • Tokyo Support [Priority 1]: Requested=139, Allocated=30, Unmet=109
```

### JSON
Detailed JSON structure for programmatic consumption.
