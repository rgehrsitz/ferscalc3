package calculation

// ProjectionBaseYear centralizes the starting calendar year for projections.
const ProjectionBaseYear = 2025

// Financial constants
const (
	DefaultDiscountRate    = 0.03 // 3% discount rate for present value calculations
	BiWeeklyPayPeriods     = 26   // Standard bi-weekly pay periods per year
	MedicareEligibilityAge = 65   // Age when Medicare eligibility begins
	SocialSecurityStartAge = 62   // Earliest Social Security claiming age
	RetirementAgeThreshold = 65   // Age for senior tax deductions
	MortalityBufferYears   = 5    // Buffer years for mortality calculations
)

// Tax-related constants
const (
	MaxInflationRate     = 0.20  // Maximum reasonable inflation rate (20%)
	MinInflationRate     = -0.10 // Minimum reasonable inflation rate (-10%)
	DefaultCOLARate      = 0.03  // Default COLA rate (3%)
	DefaultTSPReturnPre  = 0.07  // Default pre-retirement TSP return (7%)
	DefaultTSPReturnPost = 0.05  // Default post-retirement TSP return (5%)
)

// Success rate thresholds
const (
	MinSuccessRate     = 10.0  // Minimum success rate percentage
	GoodSuccessRate    = 95.0  // Good success rate threshold (95%)
	PerfectSuccessRate = 100.0 // Perfect success rate (100%)
)
