package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Record struct {
	EntryTime time.Time
	ExitTime  time.Time
	TotalWork time.Duration
}

type Employee struct {
	Name         string
	Rate         float64
	CCSS         float64
	VacationDays int // Add this field

}

// Define employees with their rates and CCSS deductions
var employees = []Employee{
	{Name: "Nayi2", Rate: 2000, CCSS: 0, VacationDays: 0},
	{Name: "Nayi", Rate: 3125, CCSS: 10000, VacationDays: 0},
	{Name: "Vero", Rate: 1300, CCSS: 0, VacationDays: 0},
	{Name: "Leidy", Rate: 2000, CCSS: (-27395 / 2), VacationDays: 1},
	{Name: "Jose Mario", Rate: 2000, CCSS: 0, VacationDays: 0},
	{Name: "Graciela", Rate: 1800, CCSS: 0, VacationDays: 0},
	{Name: "Ana", Rate: 1800, CCSS: 0, VacationDays: 0},
	{Name: "Tatiana", Rate: 1800, CCSS: 0, VacationDays: 0},
	{Name: "Angélica", Rate: 1800, CCSS: 0, VacationDays: 0},
	{Name: "Luis", Rate: 2000, CCSS: 0, VacationDays: 0},
	{Name: "Tania", Rate: 1800, CCSS: 0, VacationDays: 0},
}

// Helper maps for quick lookup
var employeeRates = make(map[string]float64)
var ccssDeductions = make(map[string]float64)

func init() {
	for _, emp := range employees {
		employeeRates[emp.Name] = emp.Rate
		ccssDeductions[emp.Name] = emp.CCSS
	}
}

var overallTotalPayment float64 // Global variable to track total payment across all files

// Aggregated data across processed files
var basePaymentPerWorker = make(map[string]float64)
var workedMinutesPerWorker = make(map[string]int)
var holidayPayPerWorker = make(map[string]float64)
var holidayMinutesPerWorker = make(map[string]int)
var holidayDatesPerWorker = make(map[string]map[string]bool)
var workedHolidayDates = make(map[string]bool)
var unworkedHolidayPayPerWorker = make(map[string]float64)
var unworkedHolidayShiftsPerWorker = make(map[string]int)
var unworkedHolidayDatesPerWorker = make(map[string]map[string]int)

// New: Map to store hours per worker per day
var hoursPerWorkerPerDay = make(map[string]map[string]float64)

func main() {
	reader := bufio.NewReader(os.Stdin)

	// Input "monto de servicio"
	fmt.Print("Ingrese el monto total de servicio a repartir: ")
	serviceAmountStr, _ := reader.ReadString('\n')
	serviceAmountStr = strings.TrimSpace(serviceAmountStr)
	serviceAmount, err := strconv.ParseFloat(serviceAmountStr, 64)
	if err != nil {
		fmt.Println("Error: el monto de servicio debe ser un número")
		return
	}

	// Ask for holidays
	var holidays []string
	fmt.Print("¿Hay días feriados? (s/n): ")
	hasHolidayStr, _ := reader.ReadString('\n')
	hasHolidayStr = strings.TrimSpace(strings.ToLower(hasHolidayStr))
	if hasHolidayStr == "s" || hasHolidayStr == "si" {
		fmt.Println("Ingrese los días feriados en formato YYYY-MM-DD, separados por coma (ejemplo: 2025-07-25,2025-08-02):")
		holidaysStr, _ := reader.ReadString('\n')
		holidaysStr = strings.TrimSpace(holidaysStr)
		if holidaysStr != "" {
			for _, h := range strings.Split(holidaysStr, ",") {
				normalized, err := normalizeHolidayDate(h)
				if err != nil {
					fmt.Printf("Formato de feriado inválido: %q (use YYYY-MM-DD o DD/MM/YYYY)\n", strings.TrimSpace(h))
					continue
				}
				holidays = append(holidays, normalized)
			}
		}
	}

	// Input vacation days for each employee
	for i := range employees {
		fmt.Printf("Ingrese días de vacaciones para %s (actual: %d): ", employees[i].Name, employees[i].VacationDays)
		vacStr, _ := reader.ReadString('\n')
		vacStr = strings.TrimSpace(vacStr)
		if vacStr != "" {
			vacDays, err := strconv.Atoi(vacStr)
			if err == nil {
				employees[i].VacationDays = vacDays
			}
		}
	}

	files, err := filepath.Glob("Report*.csv")
	if err != nil {
		fmt.Println("Error getting files:", err)
		return
	}

	if len(files) == 0 {
		fmt.Println("No matching files found")
		return
	}

	for _, file := range files {
		err := processFileWithHolidays(file, serviceAmount, holidays)
		if err != nil {
			fmt.Printf("Error processing file %s: %v\n", file, err)
		}
	}

	unworkedHolidays := findUnworkedHolidays(holidays)
	if len(unworkedHolidays) > 0 {
		fmt.Println("\nFeriados no laborados detectados (sin marcas en reloj):")
		for _, holiday := range unworkedHolidays {
			fmt.Printf("  - %s\n", holiday)
		}

		fmt.Println("\nSeleccione colaboradores a pagar en cada feriado no laborado (8 horas a tarifa normal):")
		for i, emp := range employees {
			fmt.Printf("  %2d. %s\n", i+1, emp.Name)
		}

		for _, holiday := range unworkedHolidays {
			fmt.Printf("Para el feriado %s, ingrese números de empleados separados por coma (o Enter para ninguno): ", holiday)
			selStr, _ := reader.ReadString('\n')
			selStr = strings.TrimSpace(selStr)
			if selStr == "" {
				continue
			}

			for _, part := range strings.Split(selStr, ",") {
				idx, err := strconv.Atoi(strings.TrimSpace(part))
				if err != nil || idx < 1 || idx > len(employees) {
					continue
				}
				emp := employees[idx-1]
				rate := employeeRates[emp.Name]
				unworkedHolidayPayPerWorker[emp.Name] += rate * 8
				unworkedHolidayShiftsPerWorker[emp.Name]++
				if unworkedHolidayDatesPerWorker[emp.Name] == nil {
					unworkedHolidayDatesPerWorker[emp.Name] = make(map[string]int)
				}
				unworkedHolidayDatesPerWorker[emp.Name][holiday]++
			}
		}
	}

	// Add unworked holiday pay to overall total
	for _, pay := range unworkedHolidayPayPerWorker {
		overallTotalPayment += pay
	}

	fmt.Printf("Total a pagar para todos los archivos: $%.2f\n", overallTotalPayment)

	showBarGraphAndPayments(serviceAmount)
}

// New: Map to store break minutes per worker per day
var breaksPerWorkerPerDay = make(map[string]map[string]float64)

// For break calculation between sessions
type session struct {
	entry time.Time
	exit  time.Time
	mins  int
}

// Map: worker -> date -> []session
var sessionsPerWorkerPerDay = make(map[string]map[string][]session)

func processFileWithHolidays(filename string, serviceAmount float64, holidays []string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = '|'
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("error reading CSV: %w", err)
	}

	holidaySet := make(map[string]struct{}, len(holidays))
	for _, h := range holidays {
		holidaySet[h] = struct{}{}
	}

	personWorkData := make(map[string]int)
	personPaymentData := make(map[string]float64)

	totalTiempoLaboradoAll := 0

	for i, row := range records {
		if len(row) >= 4 && i != len(records)-1 {
			if i == 0 {
				continue
			}
			colaborador := strings.TrimSpace(row[0])
			entryTime, err := parseWorkDateTime(row[1])
			if err != nil {
				fmt.Printf("No se pudo parsear hora entrada %q en fila %d: %v\n", row[1], i+1, err)
				continue
			}
			exitTime, err := parseWorkDateTime(row[2])
			if err != nil {
				fmt.Printf("No se pudo parsear hora salida %q en fila %d: %v\n", row[2], i+1, err)
				continue
			}
			formattedDate := entryTime.Format("2006-01-02")
			total := row[3]

			totalWorkDayMinutes, err := parseTotalTimeToMinutes(total)
			if err != nil {
				fmt.Println("Error parsing total time:", err)
				continue
			}

			// Store session for break calculation
			if _, ok := sessionsPerWorkerPerDay[colaborador]; !ok {
				sessionsPerWorkerPerDay[colaborador] = make(map[string][]session)
			}
			sessionsPerWorkerPerDay[colaborador][formattedDate] = append(
				sessionsPerWorkerPerDay[colaborador][formattedDate],
				session{entry: entryTime, exit: exitTime, mins: totalWorkDayMinutes},
			)

			// Store hours per worker per day
			if _, ok := hoursPerWorkerPerDay[colaborador]; !ok {
				hoursPerWorkerPerDay[colaborador] = make(map[string]float64)
			}
			hoursPerWorkerPerDay[colaborador][formattedDate] += float64(totalWorkDayMinutes) / 60.0

			// Check if this date is a holiday
			isHoliday := 0
			if _, ok := holidaySet[formattedDate]; ok {
				isHoliday = 1
			}

			payment := calculatePayment(totalWorkDayMinutes, colaborador, isHoliday)

			// Track holiday extra pay (the additional 1x premium only) and minutes separately
			if isHoliday == 1 {
				workedHolidayDates[formattedDate] = true
				holidayPayPerWorker[colaborador] += payment / 2 // only the extra premium
				holidayMinutesPerWorker[colaborador] += totalWorkDayMinutes
				if holidayDatesPerWorker[colaborador] == nil {
					holidayDatesPerWorker[colaborador] = make(map[string]bool)
				}
				holidayDatesPerWorker[colaborador][formattedDate] = true
			}

			personWorkData[colaborador] += totalWorkDayMinutes
			personPaymentData[colaborador] += payment
		}
	}

	// Calculate breaks between sessions per worker per day
	for colaborador, days := range sessionsPerWorkerPerDay {
		if _, ok := breaksPerWorkerPerDay[colaborador]; !ok {
			breaksPerWorkerPerDay[colaborador] = make(map[string]float64)
		}
		for date, sessions := range days {
			// Sort sessions by entry time
			sort.Slice(sessions, func(i, j int) bool {
				return sessions[i].entry.Before(sessions[j].entry)
			})
			breakMinutes := 0.0
			for i := 1; i < len(sessions); i++ {
				// Break is time between previous exit and current entry
				breakMin := sessions[i].entry.Sub(sessions[i-1].exit).Minutes()
				if breakMin > 0 {
					breakMinutes += breakMin
				}
			}
			breaksPerWorkerPerDay[colaborador][date] = breakMinutes
		}
	}

	totalMinutesWorkedAll := 0
	for _, minutes := range personWorkData {
		totalMinutesWorkedAll += minutes
	}

	for colaborador, totalWorkMinutes := range personWorkData {
		basePayment := personPaymentData[colaborador]
		totalHours := totalWorkMinutes / 60

		proportionalService := 0.0
		if totalMinutesWorkedAll > 0 {
			proportionalService = (float64(totalWorkMinutes) / float64(totalMinutesWorkedAll)) * serviceAmount
		}

		ccss := ccssDeductions[colaborador]

		totalPayment := basePayment + proportionalService - ccss

		basePaymentPerWorker[colaborador] += basePayment
		workedMinutesPerWorker[colaborador] += totalWorkMinutes
		overallTotalPayment += totalPayment
		totalTiempoLaboradoAll += totalHours
	}

	fmt.Printf("Total Tiempo Laborado All %dh\n", totalTiempoLaboradoAll)
	return nil
}

func normalizeHolidayDate(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("empty holiday date")
	}

	layouts := []string{"2006-01-02", "2/1/2006", "02/01/2006"}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.Format("2006-01-02"), nil
		}
	}

	return "", fmt.Errorf("invalid holiday date format")
}

func parseWorkDateTime(raw string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	layouts := []string{"2/1/2006 15:04", "02/01/2006 15:04", "2/1/2006 15:04:05", "02/01/2006 15:04:05"}

	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid datetime format")
}

func parseTotalTimeToMinutes(total string) (int, error) {
	parts := strings.Split(total, " ")

	var hours, minutes int
	var err error

	if len(parts) >= 1 {
		if strings.Contains(parts[0], "h") {
			hours, err = strconv.Atoi(strings.TrimSuffix(parts[0], "h"))
			if err != nil {
				return 0, err
			}
		}
	}

	if len(parts) == 2 {
		if strings.Contains(parts[1], "m") {
			minutes, err = strconv.Atoi(strings.TrimSuffix(parts[1], "m"))
			if err != nil {
				return 0, err
			}
		}
	}

	// fmt.Printf("Parsing Time: %s = %d hours, %d minutes\n", total, hours, minutes)
	return hours*60 + minutes, nil
}

func calculatePayment(totalWorkMinutes int, colaborador string, isHoliday int) float64 {
	rate := employeeRates[colaborador]
	hourlyPay := rate / 60

	if isHoliday == 1 {
		hourlyPay = (rate * 2) / 60 // 2x rate per hour, divided by 60 for per minute
	}

	return float64(totalWorkMinutes) * hourlyPay
}

func showBarGraphAndPayments(serviceAmount float64) {
	fmt.Println("\nResumen de horas trabajadas, descansos y pagos por colaborador (ordenados):")
	fmt.Printf("Monto total de servicio a repartir: $%.2f\n", serviceAmount)
	fmt.Println("-------------------------------------------------------------------")

	// Get sorted list of employee names (worked hours + any unworked holiday pay)
	var employeeNames []string
	nameSet := make(map[string]struct{})
	for name := range hoursPerWorkerPerDay {
		nameSet[name] = struct{}{}
	}
	for name := range unworkedHolidayPayPerWorker {
		nameSet[name] = struct{}{}
	}
	for name := range nameSet {
		employeeNames = append(employeeNames, name)
	}
	sort.Strings(employeeNames)

	// Calculate total minutes worked by all employees (excluding vacation days)
	totalMinutesWorkedAll := 0
	for _, emp := range employeeNames {
		empObj := getEmployeeByName(emp)
		vacDays := 0
		if empObj != nil {
			vacDays = empObj.VacationDays
		}
		vacMinutes := vacDays * 8 * 60
		empMinutes := 0
		for _, mins := range hoursPerWorkerPerDay[emp] {
			empMinutes += int(mins * 60)
		}
		if empMinutes < vacMinutes {
			vacMinutes = empMinutes // Prevent negative
		}
		totalMinutesWorkedAll += empMinutes - vacMinutes
	}

	for _, colaborador := range employeeNames {
		fmt.Printf("%s:\n", colaborador)
		days := hoursPerWorkerPerDay[colaborador]

		// Sort the dates for this worker
		var dates []string
		for date := range days {
			dates = append(dates, date)
		}
		sort.Strings(dates)

		totalHours := 0.0
		totalBreak := 0.0
		totalMinutes := 0

		for _, date := range dates {
			hours := days[date]
			breakMin := breaksPerWorkerPerDay[colaborador][date]
			totalHours += hours
			totalBreak += breakMin
			totalMinutes += int(hours * 60)

			holidayMarker := ""
			if holidayDatesPerWorker[colaborador][date] {
				holidayMarker = " [FERIADO]"
			}

			fmt.Printf("  %s | %6.2f h  | Descanso total: %2.0f min%s\n", date, hours, breakMin, holidayMarker)
		}

		// Vacation calculation
		empObj := getEmployeeByName(colaborador)
		vacDays := 0
		if empObj != nil {
			vacDays = empObj.VacationDays
		}
		vacHours := float64(vacDays) * 8.0
		vacMinutes := vacDays * 8 * 60

		fmt.Printf("  TOTAL HORAS: %5.2f h | TOTAL DESCANSO: %.0f min\n", totalHours, totalBreak)
		if vacDays > 0 {
			fmt.Printf("  DÍAS DE VACACIONES: %d (%.2f h no cuentan para servicio)\n", vacDays, vacHours)
		}

		// Payment summary for this employee
		rate := employeeRates[colaborador]
		vacationPay := float64(vacMinutes) * rate / 60

		workedPay, ok := basePaymentPerWorker[colaborador]
		if !ok {
			workedPay = float64(totalMinutes-vacMinutes) * rate / 60
			if workedPay < 0 {
				workedPay = 0
			}
		}

		normalHours := float64(totalMinutes) / 60.0
		basePay := workedPay + vacationPay
		ccss := ccssDeductions[colaborador]

		// Service is only for worked minutes (not vacation)
		serviceMinutes := totalMinutes - vacMinutes
		if serviceMinutes < 0 {
			serviceMinutes = 0
		}
		proportionalService := 0.0
		if totalMinutesWorkedAll > 0 {
			proportionalService = (float64(serviceMinutes) / float64(totalMinutesWorkedAll)) * serviceAmount
		}
		totalPayment := basePay + proportionalService - ccss

		holidayExtraPay := holidayPayPerWorker[colaborador]
		holidayMins := holidayMinutesPerWorker[colaborador]
		// normalWorkedPay includes holiday hours at normal rate; holidayExtraPay is only the extra premium
		normalWorkedPay := workedPay - holidayExtraPay

		if vacDays > 0 {
			fmt.Printf("  Monto h. normales:      $%.2f\n", normalWorkedPay)
			fmt.Printf("  Monto vacaciones:       $%.2f\n", vacationPay)
		} else {
			fmt.Printf("  Monto h. normales:      $%.2f\n", normalWorkedPay)
		}
		if holidayMins > 0 {
			fmt.Printf("  Recargo feriados:       $%.2f  (%.2f h x tarifa extra)\n", holidayExtraPay, float64(holidayMins)/60.0)
		}
		if unworkedPay := unworkedHolidayPayPerWorker[colaborador]; unworkedPay > 0 {
			shifts := unworkedHolidayShiftsPerWorker[colaborador]
			fmt.Printf("  Feriado no laborado:    $%.2f  (%d turno(s) de 8 h)\n", unworkedPay, shifts)
			detail := formatUnworkedHolidayDetail(colaborador)
			if detail != "" {
				fmt.Printf("  FERIADO no laborado:   %s\n", detail)
			}
			basePay += unworkedPay
			totalPayment += unworkedPay
		}
		fmt.Printf("  Tiempo pagado: %.2f h | Monto base: $%.2f\n", normalHours, basePay)
		fmt.Printf("  Servicio: $%.2f | CCSS: $%.2f | TOTAL: $%.2f\n\n",
			proportionalService, ccss, totalPayment)
	}
}

func getEmployeeByName(name string) *Employee {
	for i := range employees {
		if employees[i].Name == name {
			return &employees[i]
		}
	}
	return nil
}

func findUnworkedHolidays(holidays []string) []string {
	seen := make(map[string]bool)
	var unworked []string
	for _, holiday := range holidays {
		if seen[holiday] {
			continue
		}
		seen[holiday] = true
		if !workedHolidayDates[holiday] {
			unworked = append(unworked, holiday)
		}
	}
	sort.Strings(unworked)
	return unworked
}

func formatUnworkedHolidayDetail(worker string) string {
	dateCounts, ok := unworkedHolidayDatesPerWorker[worker]
	if !ok || len(dateCounts) == 0 {
		return ""
	}

	var dates []string
	for date := range dateCounts {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	parts := make([]string, 0, len(dates))
	for _, date := range dates {
		count := dateCounts[date]
		if count > 1 {
			parts = append(parts, fmt.Sprintf("%s x%d", date, count))
			continue
		}
		parts = append(parts, date)
	}

	return strings.Join(parts, ", ")
}

// Helper to get sessions for a worker and date
func sessionsFor(worker, date string) []struct{ entry, exit time.Time } {
	sessions := []struct{ entry, exit time.Time }{}
	if workerSessions, ok := sessionsPerWorkerPerDay[worker]; ok {
		if daySessions, ok := workerSessions[date]; ok {
			for _, s := range daySessions {
				sessions = append(sessions, struct{ entry, exit time.Time }{s.entry, s.exit})
			}
		}
	}
	return sessions
}
