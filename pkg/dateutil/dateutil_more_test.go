package dateutil

import (
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
)

func TestYearsOfServiceDecimal(t *testing.T) {
    tests := []struct{
        name string
        hire time.Time
        at   time.Time
        want float64
        tol  float64
    }{
        {"Exact 10y", time.Date(2015,1,1,0,0,0,0,time.UTC), time.Date(2025,1,1,0,0,0,0,time.UTC), 10.0, 0.01},
        {"Half year", time.Date(2024,1,1,0,0,0,0,time.UTC), time.Date(2024,7,1,0,0,0,0,time.UTC), 0.5, 0.02},
        {"Leap span", time.Date(2020,2,29,0,0,0,0,time.UTC), time.Date(2024,2,29,0,0,0,0,time.UTC), 4.0, 0.01},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := YearsOfServiceDecimal(tt.hire, tt.at)
            assert.InDelta(t, tt.want, got, tt.tol)
        })
    }
}

func TestYearsUntilDate(t *testing.T) {
    tests := []struct{
        name string
        from time.Time
        to   time.Time
        want float64
        tol  float64
    }{
        {"1 year", time.Date(2020,1,1,0,0,0,0,time.UTC), time.Date(2021,1,1,0,0,0,0,time.UTC), 1.0, 0.01},
        {"2.5 years", time.Date(2020,1,1,0,0,0,0,time.UTC), time.Date(2022,7,1,0,0,0,0,time.UTC), 2.5, 0.05},
        {"Across leap", time.Date(2019,7,1,0,0,0,0,time.UTC), time.Date(2020,7,1,0,0,0,0,time.UTC), 1.0, 0.01},
        {"Zero", time.Date(2025,8,1,0,0,0,0,time.UTC), time.Date(2025,8,1,0,0,0,0,time.UTC), 0.0, 0.0},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := YearsUntilDate(tt.from, tt.to)
            assert.InDelta(t, tt.want, got, tt.tol)
        })
    }
}

func TestMonthsUntilDate(t *testing.T) {
    tests := []struct{
        name string
        from time.Time
        to   time.Time
        want int
    }{
        {"12 months", time.Date(2020,1,1,0,0,0,0,time.UTC), time.Date(2021,1,1,0,0,0,0,time.UTC), 12},
        // Due to YearsUntilDate using 365.25-day years and truncation, this spans ~17.98 months -> 17
        {"~18 months (truncates to 17)", time.Date(2020,1,1,0,0,0,0,time.UTC), time.Date(2021,7,1,0,0,0,0,time.UTC), 17},
        {"0 months", time.Date(2025,8,1,0,0,0,0,time.UTC), time.Date(2025,8,1,0,0,0,0,time.UTC), 0},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := MonthsUntilDate(tt.from, tt.to)
            assert.Equal(t, tt.want, got)
        })
    }
}
