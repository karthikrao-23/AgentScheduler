# Agent Scheduler System Architecture

## 1. Overview
Agent Scheduler is a high-performance command-line tool designed to calculate and schedule support agent requirements across multiple timezones. It translates raw call volume data into precise, hourly staffing requirements, accounting for complex variables like partial start times, capacity constraints, business priorities, and variable call durations.

## 2. Problem Statement
Support organizations need to translate forecasted call volumes (e.g., "100 calls between 9:30 AM and 11:45 AM") into concrete staffing schedules (e.g., "Need 12 agents at 9:00, 15 agents at 10:00"). This is challenging due to:
-   **Partial Hours**: Shifts rarely align perfectly with hourly boundaries.
-   **Timezones**: Inputs come in various mixed local times (EST, PST, Tokyo).
-   **Capacity Limits**: There is often a hard cap on available agents per hour.
-   **Prioritization**: When demand exceeds supply, high-value clients must be staffed first.

## 3. System Design

### 3.1 Core Components

```
graph TD
    Input[CSV Input] --> Parser
    Parser -->|CallData[]| Scheduler
    Scheduler -->|Schedule Model| Formatter
    Formatter -->|Output| Text/JSON/CSV
    
    subgraph Observability
    Parser -.-> Metrics
    Scheduler -.-> Metrics
    end
```

1.  **Parser (`parser/`)**:
    -   Ingests CSV data.
    -   Handles timezone normalization (converting all times to a comparable timeline).
    -   Validates data integrity (start < end, non-negative values).

2.  **Scheduler (`scheduler/`)**:
    -   The core logic engine.
    -   Converts "calls over a duration" into "agents per specific hour".
    -   Applies capacity constraints and utilization multipliers.
    -   Implements priority-based resource smoothing.

3.  **Formatter (`formatter/`)**:
    -   Decouples internal data structures from output presentation.
    -   Supports multiple formats (Text for humans, JSON/CSV for machines).

4.  **Observability (`metrics/`)**:
    -   Provides operational transparency via Prometheus metrics.

### 3.2 Key Algorithms

#### 3.2.1 Proportional Hourly Allocation
How do we convert "100 calls from 9:30 to 11:00" into hourly needs?
1.  **Calculate Calls/Hour**: `Rate = Total Calls / Total Duration (1.5h) = 66.6 calls/hr`.
2.  **Segment by Hour**:
    -   **09:00 - 10:00**: Overlap is 9:30-10:00 (0.5h). `Demand = 66.6 * 0.5 = 33.3 calls`.
    -   **10:00 - 11:00**: Overlap is full hour (1.0h). `Demand = 66.6 * 1.0 = 66.6 calls`.
3.  **Convert to Agents**: `Agents = Ceil(Demand * AHT / 3600)`.

#### 3.2.2 Priority-Aware Smoothing (Bin Packing)
When `Total Demand > Capacity`, how do we decide who gets staff?
1.  **Sort**: Sort all requests for a given hour by Priority (1 = Highest).
    -   *Tie-Breaker*: If priorities are equal, sort alphabetically by Customer Name to ensure deterministic scheduling.
2.  **Pass 1 (Full Fill)**: Iterate top-down. If `Remaining Capacity >= Request`, fully satisfy it.
3.  **Pass 2 (Partial Fill)**: If `Remaining Capacity > 0` but less than request, give the remainder to the next highest priority client.
4.  **Record Unmet**: Track exactly which clients lost coverage for reporting.

> **Note on Capacity Definition**:
> In this system, "Capacity" refers to **Per-Hour Concurrent Headcount** (e.g., "500 seats available in the call center").
> It does *not* refer to "Total Daily Agent-Hours" (Budget). The constraint is applied independently to each hour slot.

### 3.3 Data Structures

**`CallData` (Input)**
-   Normalized Start/End times (timezone aware).
-   Priority, Efficiency (Util multiplier).

**`Schedule` (Output)**
-   `HourlyRequirements`: Map for Hour -> List of Assignments.
-   `UnmetDemands`: Detailed breakdown of capacity breaches.

## 4. Observability & Metrics

We utilize a **Custom Prometheus Registry** to expose business-critical data without polluting the output with Go runtime metrics.

### 4.1 Strategy
-   **Pull Model**: For local debugging, use `-wait` flag to keep the process alive for scraping.
-   **Push Model**: For production Batch jobs, use `-push-url` to flush final metrics to a Pushgateway.

### 4.2 Key Metrics
| Metric | Purpose |
|--------|---------|
| `scheduler_agents_unmet_total` | **Critical**. Direct measure of lost business opportunity. |
| `scheduler_high_priority_unsatisfied_total` | **Critical**. Counts times a VIP client got 0 agents. |
| `parser_errors_total` | **Operational**. Indicates bad input data quality. |

## 5. Future Enhancements
-   **Shift Optimization**: Currently generates requirements per hour. Future versions could group these into 8-hour shifts.
-   **Capacity Management**: Future versions could support dynamic capacity changes, allowing for more flexible staffing models.
-   **UI/UX**: Future versions could support a web-based UI for visualizing the schedule and capacity planning.