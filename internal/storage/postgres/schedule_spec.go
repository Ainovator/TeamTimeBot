package postgres

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

type ScheduleSpec struct {
	Weekdays []int // 1=Mon ... 7=Sun
	SendAt   string
}

func BuildScheduleExpr(weekdays []int, sendAt string) (string, error) {
	if _, err := time.Parse("15:04", sendAt); err != nil {
		return "", errors.New("invalid time format, use HH:MM")
	}

	if len(weekdays) == 0 {
		weekdays = []int{1, 2, 3, 4, 5, 6, 7}
	}

	uniq := make([]int, 0, len(weekdays))
	seen := map[int]bool{}
	for _, day := range weekdays {
		if day < 1 || day > 7 {
			return "", errors.New("weekday must be between 1 and 7")
		}
		if !seen[day] {
			seen[day] = true
			uniq = append(uniq, day)
		}
	}

	slices.Sort(uniq)
	values := make([]string, 0, len(uniq))
	for _, day := range uniq {
		values = append(values, strconv.Itoa(day))
	}

	return strings.Join(values, ",") + "|" + sendAt, nil
}

func ParseScheduleExpr(expr string) (ScheduleSpec, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return ScheduleSpec{}, errors.New("schedule expression is empty")
	}

	if _, err := time.Parse("15:04", expr); err == nil {
		return ScheduleSpec{
			Weekdays: []int{1, 2, 3, 4, 5, 6, 7},
			SendAt:   expr,
		}, nil
	}

	parts := strings.Split(expr, "|")
	if len(parts) != 2 {
		return ScheduleSpec{}, errors.New("invalid schedule expression")
	}

	weekdaysCSV := strings.TrimSpace(parts[0])
	sendAt := strings.TrimSpace(parts[1])
	if _, err := time.Parse("15:04", sendAt); err != nil {
		return ScheduleSpec{}, errors.New("invalid time format, use HH:MM")
	}

	weekdays, err := ParseWeekdaysCSV(weekdaysCSV)
	if err != nil {
		return ScheduleSpec{}, err
	}

	return ScheduleSpec{
		Weekdays: weekdays,
		SendAt:   sendAt,
	}, nil
}

func ParseWeekdaysCSV(value string) ([]int, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "*" {
		return []int{1, 2, 3, 4, 5, 6, 7}, nil
	}

	parts := strings.Split(value, ",")
	weekdays := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		day, err := strconv.Atoi(part)
		if err != nil || day < 1 || day > 7 {
			return nil, errors.New("weekdays must be numbers 1..7")
		}

		weekdays = append(weekdays, day)
	}

	if len(weekdays) == 0 {
		return nil, errors.New("weekdays are empty")
	}

	return weekdays, nil
}

func FormatWeekdays(weekdays []int) string {
	labels := map[int]string{
		1: "пн",
		2: "вт",
		3: "ср",
		4: "чт",
		5: "пт",
		6: "сб",
		7: "вс",
	}

	result := make([]string, 0, len(weekdays))
	for _, day := range weekdays {
		label, ok := labels[day]
		if !ok {
			label = strconv.Itoa(day)
		}
		result = append(result, label)
	}

	return strings.Join(result, ",")
}

func ComputeNextRunAt(timezone, scheduleExpr string, nowUTC time.Time) (time.Time, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timezone %q: %w", timezone, err)
	}

	spec, err := ParseScheduleExpr(scheduleExpr)
	if err != nil {
		return time.Time{}, err
	}

	parsedTime, _ := time.Parse("15:04", spec.SendAt)
	nowLocal := nowUTC.In(location)

	for addDays := 0; addDays <= 7; addDays++ {
		candidateDate := nowLocal.AddDate(0, 0, addDays)
		weekday := toISOWeekday(candidateDate.Weekday())
		if !slices.Contains(spec.Weekdays, weekday) {
			continue
		}

		candidate := time.Date(
			candidateDate.Year(),
			candidateDate.Month(),
			candidateDate.Day(),
			parsedTime.Hour(),
			parsedTime.Minute(),
			0,
			0,
			location,
		)

		if candidate.After(nowLocal) {
			return candidate.UTC(), nil
		}
	}

	return time.Time{}, errors.New("failed to compute next run")
}

func FormatScheduleExprForDisplay(expr string) string {
	spec, err := ParseScheduleExpr(expr)
	if err != nil {
		return expr
	}
	return fmt.Sprintf("%s в %s", FormatWeekdays(spec.Weekdays), spec.SendAt)
}

func toISOWeekday(weekday time.Weekday) int {
	if weekday == time.Sunday {
		return 7
	}
	return int(weekday)
}
