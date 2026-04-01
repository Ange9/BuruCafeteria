package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Employee represents a cafeteria worker with their pay configuration
type Employee struct {
	Name         string
	Rate         float64
	CCSS         float64
	VacationDays int
}

type session struct {
	entry time.Time
	exit  time.Time
	mins  int
}

var employees = []Employee{
	{Name: "Nayi1", Rate: 2000, CCSS: 0},
	{Name: "Nayi", Rate: 3125, CCSS: 0},
	{Name: "Vero", Rate: 1300, CCSS: 0},
	{Name: "Leidy", Rate: 2000, CCSS: -27395.0 / 2},
	{Name: "Jose Mario", Rate: 2000, CCSS: 0},
	{Name: "Graciela", Rate: 1800, CCSS: 0},
	{Name: "Ana", Rate: 1800, CCSS: 0},
	{Name: "Tatiana", Rate: 1800, CCSS: 0},
	{Name: "Angélica", Rate: 1800, CCSS: 0},
	{Name: "Luis", Rate: 2000, CCSS: 0},
	{Name: "Tania", Rate: 1800, CCSS: 0},
}

var (
	employeeRates           = make(map[string]float64)
	ccssDeductions          = make(map[string]float64)
	hoursPerWorkerPerDay    = make(map[string]map[string]float64)
	breaksPerWorkerPerDay   = make(map[string]map[string]float64)
	sessionsPerWorkerPerDay = make(map[string]map[string][]session)
	overallTotalPayment     float64
)

func init() {
	rebuildEmployeeMaps()
}

func rebuildEmployeeMaps() {
	employeeRates = make(map[string]float64)
	ccssDeductions = make(map[string]float64)
	for _, emp := range employees {
		employeeRates[emp.Name] = emp.Rate
		ccssDeductions[emp.Name] = emp.CCSS
	}
}

func resetPayrollState() {
	hoursPerWorkerPerDay = make(map[string]map[string]float64)
	breaksPerWorkerPerDay = make(map[string]map[string]float64)
	sessionsPerWorkerPerDay = make(map[string]map[string][]session)
	overallTotalPayment = 0
	rebuildEmployeeMaps()
}

// runPayroll processes all CSV files and returns a formatted report
func runPayroll(csvFiles []string, serviceAmount float64, holidays []string) string {
	var out strings.Builder
	resetPayrollState()

	for _, file := range csvFiles {
		err := processCSV(file, serviceAmount, holidays, &out)
		if err != nil {
			fmt.Fprintf(&out, "Error procesando %s: %v\n", file, err)
		}
	}

	fmt.Fprintf(&out, "\nTotal a pagar para todos los archivos: ₡%.2f\n\n", overallTotalPayment)
	generateReport(serviceAmount, &out)
	return out.String()
}

func processCSV(filename string, serviceAmount float64, holidays []string, out *strings.Builder) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("error abriendo archivo: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = '|'
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("error leyendo CSV: %w", err)
	}

	personWorkData := make(map[string]int)
	personPaymentData := make(map[string]float64)

	for i, row := range records {
		if len(row) >= 4 && i != len(records)-1 {
			if i == 0 {
				continue // skip header
			}
			colaborador := strings.TrimSpace(row[0])
			entryTime, _ := time.Parse("2/1/2006 15:04", row[1])
			exitTime, _ := time.Parse("2/1/2006 15:04", row[2])
			formattedDate := entryTime.Format("2006-01-02")
			total := row[3]

			totalWorkDayMinutes, err := parseTotalTimeToMinutes(total)
			if err != nil {
				fmt.Fprintf(out, "Error parseando tiempo: %v\n", err)
				continue
			}

			// Store session
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

			// Check if holiday
			isHoliday := 0
			for _, h := range holidays {
				if formattedDate == h {
					isHoliday = 1
					fmt.Fprintf(out, "Fecha %s es feriado para %s\n", formattedDate, colaborador)
					break
				}
			}

			payment := calculatePayment(totalWorkDayMinutes, colaborador, isHoliday)
			personWorkData[colaborador] += totalWorkDayMinutes
			personPaymentData[colaborador] += payment
		}
	}

	// Calculate breaks between sessions
	for colaborador, days := range sessionsPerWorkerPerDay {
		if _, ok := breaksPerWorkerPerDay[colaborador]; !ok {
			breaksPerWorkerPerDay[colaborador] = make(map[string]float64)
		}
		for date, sess := range days {
			sort.Slice(sess, func(i, j int) bool {
				return sess[i].entry.Before(sess[j].entry)
			})
			breakMinutes := 0.0
			for k := 1; k < len(sess); k++ {
				bm := sess[k].entry.Sub(sess[k-1].exit).Minutes()
				if bm > 0 {
					breakMinutes += bm
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
		ccss := ccssDeductions[colaborador]
		proportionalService := 0.0
		if totalMinutesWorkedAll > 0 {
			proportionalService = (float64(totalWorkMinutes) / float64(totalMinutesWorkedAll)) * serviceAmount
		}
		totalPayment := basePayment + proportionalService - ccss
		overallTotalPayment += totalPayment
	}

	fmt.Fprintf(out, "Archivo %s procesado.\n", filename)
	return nil
}

func generateReport(serviceAmount float64, out *strings.Builder) {
	fmt.Fprintln(out, "══════════════════════════════════════════════════════════════")
	fmt.Fprintln(out, "  RESUMEN DE PLANILLA")
	fmt.Fprintf(out, "  Monto de servicio a repartir: ₡%.2f\n", serviceAmount)
	fmt.Fprintln(out, "══════════════════════════════════════════════════════════════")

	var employeeNames []string
	for name := range hoursPerWorkerPerDay {
		employeeNames = append(employeeNames, name)
	}
	sort.Strings(employeeNames)

	// Total minutes worked (excluding vacation) for service distribution
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
			vacMinutes = empMinutes
		}
		totalMinutesWorkedAll += empMinutes - vacMinutes
	}

	for _, colaborador := range employeeNames {
		fmt.Fprintf(out, "\n── %s ──\n", colaborador)
		days := hoursPerWorkerPerDay[colaborador]

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
			fmt.Fprintf(out, "  %s  |  %6.2f h  |  Descanso: %2.0f min\n", date, hours, breakMin)
		}

		empObj := getEmployeeByName(colaborador)
		vacDays := 0
		if empObj != nil {
			vacDays = empObj.VacationDays
		}
		vacHours := float64(vacDays) * 8.0
		vacMinutes := vacDays * 8 * 60

		fmt.Fprintf(out, "  TOTAL HORAS: %5.2f h  |  TOTAL DESCANSO: %.0f min\n", totalHours, totalBreak)
		if vacDays > 0 {
			fmt.Fprintf(out, "  DÍAS DE VACACIONES: %d (%.2f h)\n", vacDays, vacHours)
		}

		workedNormalMinutes := totalMinutes
		workedExtraMinutes := 0
		if totalMinutes > 96*600 {
			workedNormalMinutes = 96 * 60
			workedExtraMinutes = totalMinutes - workedNormalMinutes
		}

		vacationNormalMinutes := vacMinutes
		rate := employeeRates[colaborador]

		workedPay := float64(workedNormalMinutes)*rate/60 + float64(workedExtraMinutes)*rate/60*1.5
		vacationPay := float64(vacationNormalMinutes) * rate / 60
		ccss := ccssDeductions[colaborador]

		serviceMinutes := totalMinutes
		proportionalService := 0.0
		if totalMinutesWorkedAll > 0 {
			proportionalService = (float64(serviceMinutes) / float64(totalMinutesWorkedAll)) * serviceAmount
		}
		totalPayment := workedPay + vacationPay + proportionalService - ccss

		normalHours := float64(workedNormalMinutes+vacationNormalMinutes) / 60.0
		normalPay := float64(workedNormalMinutes+vacationNormalMinutes) * rate / 60

		if vacDays > 0 {
			fmt.Fprintf(out, "  Monto por días trabajados: ₡%.2f\n", workedPay)
			fmt.Fprintf(out, "  Monto por vacaciones:      ₡%.2f\n", vacationPay)
		} else {
			fmt.Fprintf(out, "  Monto por días trabajados: ₡%.2f\n", workedPay)
		}
		fmt.Fprintf(out, "  Tiempo normal: %.2f h  |  Monto normal: ₡%.2f\n", normalHours, normalPay)
		fmt.Fprintf(out, "  Servicio: ₡%.2f  |  CCSS: ₡%.2f  |  TOTAL: ₡%.2f\n", proportionalService, ccss, totalPayment)
	}

	fmt.Fprintln(out, "\n══════════════════════════════════════════════════════════════")
}

func parseTotalTimeToMinutes(total string) (int, error) {
	parts := strings.Split(total, " ")
	var hours, minutes int
	var err error

	if len(parts) >= 1 && strings.Contains(parts[0], "h") {
		hours, err = strconv.Atoi(strings.TrimSuffix(parts[0], "h"))
		if err != nil {
			return 0, err
		}
	}
	if len(parts) == 2 && strings.Contains(parts[1], "m") {
		minutes, err = strconv.Atoi(strings.TrimSuffix(parts[1], "m"))
		if err != nil {
			return 0, err
		}
	}
	return hours*60 + minutes, nil
}

func calculatePayment(totalWorkMinutes int, colaborador string, isHoliday int) float64 {
	rate := employeeRates[colaborador]
	hourlyPay := rate / 60
	if isHoliday == 1 {
		hourlyPay = (rate * 2) / 60
	}
	return float64(totalWorkMinutes) * hourlyPay
}

func getEmployeeByName(name string) *Employee {
	for i := range employees {
		if employees[i].Name == name {
			return &employees[i]
		}
	}
	return nil
}
