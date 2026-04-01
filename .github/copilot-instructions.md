# Copilot Instructions for BuruCafeteria

## Project Overview

BuruCafeteria is a **payroll and employee timesheet management CLI** for a cafeteria operation in Costa Rica. It processes employee time-tracking CSV reports to calculate wages, CCSS (Caja Costarricense de Seguro Social) deductions, holiday pay, vacation pay, extra-time pay, and service charge (tips) distribution.

## Tech Stack

- **Language:** Go 1.21+
- **Dependencies:** Standard library only (the Fyne GUI dependency in `go.mod` is unused)
- **Data format:** Pipe-delimited (`|`) CSV files exported from a time-tracking system
- **No database** — all data comes from CSV files and hardcoded employee definitions

## Project Structure

```
BuruCafeteria/
├── main.go          # All application logic (single-file monolith)
├── Archive/         # Historical CSV report files
├── go.mod / go.sum  # Go module files
└── README.md
```

## Coding Conventions

### Language & Naming
- **Variables and functions:** camelCase (`processFileWithHolidays`, `employeeRates`)
- **Types:** PascalCase (`Employee`, `Record`); lowercase for unexported (`session`)
- **Domain terms mix Spanish and English:**
  - Spanish: `colaborador` (employee), `descanso` (break), `feriado` (holiday), `vacaciones` (vacation), `servicio` (service charge)
  - Use Spanish for user-facing text (prompts, output labels)
  - Use English for code identifiers and comments

### Error Handling
- Use `fmt.Errorf("context: %w", err)` for wrapping errors with context
- Validate file operations and return errors to callers
- Time parsing currently uses blank identifier `_` for errors — when modifying parsing code, prefer proper error handling

### Code Style
- Keep comments concise and only where logic is non-obvious
- Use `init()` for populating lookup maps from the employee slice
- Date format for display: `2006-01-02` (ISO) and `03:04 PM` (12-hour)
- Date parsing from CSV: `2/1/2006 15:04` (D/M/YYYY 24-hour, as used in Costa Rica)

## Domain Logic

### Payment Calculation Rules
1. **Base pay** = worked_minutes × (hourly_rate / 60)
2. **Holiday pay** = worked_minutes × (hourly_rate × 2 / 60) — holidays use a 2× multiplier
3. **Vacation pay** = vacation_days × 8 hours × hourly_rate — always at normal rate
4. **Extra time** = minutes over daily threshold × (hourly_rate × 1.5 / 60)
5. **Service distribution** = (employee_minutes / total_minutes) × service_amount — proportional
6. **CCSS deduction** = per-employee fixed deduction amount
7. **Total** = base_pay + holiday_pay + vacation_pay + extra_time + service_distribution − CCSS

### CSV Format
- Pipe-delimited (`|`), first row is header, last row is footer (both skipped)
- Columns: `Vendedor | Hora entrada | Hora salida | Total`
- Time column format: `D/M/YYYY HH:MM`
- Total column format: `Xh Ym`
- Filenames follow: `Report_1_YYMMDDHHMMSS.csv`

### Employee Data
- Employees are hardcoded in a global `employees` slice with Name, Rate, CCSS, and VacationDays
- Lookup maps (`employeeRates`, `ccssDeductions`) are populated in `init()`
- When adding or removing employees, update both the `employees` slice and ensure `init()` rebuilds the maps

## Key Guidelines for Copilot

- **Currency:** All monetary values are in Costa Rican Colones (₡ / CRC). Do not assume USD.
- **Locale:** User interaction is in **Spanish**. Generate user-facing strings in Spanish.
- **Time zone:** Assume Costa Rica time (CST, UTC-6). No daylight saving time.
- **Do not remove** the `Archive/` directory or its CSV files — they are historical payroll records.
- **Testing:** No tests exist yet. When writing new tests, use Go's standard `testing` package.
- **Keep it simple:** This is a small internal tool. Avoid over-engineering or adding unnecessary abstractions.
