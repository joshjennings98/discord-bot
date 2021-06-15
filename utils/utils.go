package utils

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	commonerrors "github.com/joshjennings98/discord-bot/errors"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	SimpleTimeFormat = "15:04:05"
	FullDateFormat   = "02/01/06 03:04:05 PM"
)

type Validator interface {
	Validate() error
}

// Use reflection to find embedded structs and validate them
func ValidateEmbedded(cfg Validator) error {
	r := reflect.ValueOf(cfg).Elem()
	for i := 0; i < r.NumField(); i++ {
		f := r.Field(i)
		if f.Kind() == reflect.Struct {
			validator, ok := f.Addr().Interface().(Validator)
			if !ok {
				continue
			}
			err := validator.Validate()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Loads the configuration from the environment and puts the entries into the configuration object.
// If not found in the environment, the values will come from the default values.
// `envVarPrefix` defines a prefix that ENVIRONMENT variables will use.  E.g. if your prefix is "spf", the env registry will look for env variables that start with "SPF_".
func Load(envVarPrefix string, configurationToSet Validator, defaultConfiguration Validator) error {
	return LoadFromViper(viper.New(), envVarPrefix, configurationToSet, defaultConfiguration)

}

// Same as `Load` but instead of creating a new viper session, reuse the one provided.
func LoadFromViper(viperSession *viper.Viper, envVarPrefix string, configurationToSet Validator, defaultConfiguration Validator) (err error) {
	// Load Defaults
	var defaults map[string]interface{}
	err = mapstructure.Decode(defaultConfiguration, &defaults)
	if err != nil {
		return
	}
	err = viperSession.MergeConfigMap(defaults)
	if err != nil {
		return
	}

	// Load .env file contents into environment, if it exists
	_ = godotenv.Load(".env")

	// Load Environment variables
	viperSession.SetEnvPrefix(envVarPrefix)
	viperSession.AllowEmptyEnv(false)
	viperSession.AutomaticEnv()
	viperSession.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	// Merge together all the sources and unmarshal into struct
	if err := viperSession.Unmarshal(configurationToSet); err != nil {
		return fmt.Errorf("unable to decode config into struct, %w", err)
	}
	// Run validation
	err = configurationToSet.Validate()
	return
}

// Binds pflags to environment variable.
func BindFlagToEnv(viperSession *viper.Viper, envVarPrefix string, envVar string, flag *pflag.Flag) (err error) {
	err = viperSession.BindPFlag(envVar, flag)
	if err != nil {
		return
	}
	trimmed := strings.TrimPrefix(strings.TrimPrefix(strings.ToLower(envVar), strings.ToLower(envVarPrefix)), "_")
	err = viperSession.BindPFlag(trimmed, flag)
	if err != nil {
		return
	}
	err = viperSession.BindPFlag(strings.ReplaceAll(trimmed, "_", "."), flag)
	if err != nil {
		return
	}
	err = viperSession.BindPFlag(strings.ReplaceAll(envVar, "_", "."), flag)
	return
}

func IsValidDate(s string) bool {
	re := regexp.MustCompile(`^(3[01]|[12][0-9]|0?[1-9])/(1[0-2]|0?[1-9])`)
	return re.MatchString(s)
}

func RemoveChars(s string, chars []string) (newS string) {
	newS = s
	for _, char := range chars {
		newS = strings.ReplaceAll(newS, char, "")
	}
	return
}

func InHourInterval(n int, timeToCheck time.Time) bool {
	startTime := fmt.Sprintf("%s:00:00", AppendZero(n))
	endTime := fmt.Sprintf("%s:00:00", AppendZero(n+1))

	start, err := time.Parse(SimpleTimeFormat, startTime)
	if err != nil {
		return false
	}
	end, err := time.Parse(SimpleTimeFormat, endTime)
	if err != nil {
		return false
	}
	check, err := time.Parse(SimpleTimeFormat, timeToCheck.Format(SimpleTimeFormat))
	if err != nil {
		return false
	}

	return !check.Before(start) && !check.After(end)
}

func SplitCommand(input string) []string {
	s := strings.Split(input, " ")
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}

type Month struct {
	Name string
	Days int
}

func AppendZero(i int) string {
	if i < 10 {
		return fmt.Sprintf("0%d", i)
	}
	return fmt.Sprintf("%d", i)
}

func ConvertYearDayToDate(day string) (date string, err error) {
	count, err := strconv.Atoi(day)
	if err != nil {
		return "", fmt.Errorf("error parsing day as date %w", err)
	}
	monthDays := [12]Month{
		{"January", 31},
		{"February", 28},
		{"March", 31},
		{"April", 30},
		{"May", 31},
		{"June", 30},
		{"July", 31},
		{"August", 31},
		{"September", 30},
		{"October", 31},
		{"November", 30},
		{"December", 30},
	}
	for _, month := range monthDays {
		count -= month.Days
		if count < 0 {
			day := count + month.Days
			date := fmt.Sprintf("%s %d", month.Name, day)
			return date, nil
		}
	}
	return "", fmt.Errorf("error parsing day as date %w", err)
}

func DaysInThisYear() int {
	y := time.Now().Year()
	if y%4 == 0 {
		return 366
	}
	return 365
}

func Contains(arr interface{}, elem interface{}) bool {
	arrV := reflect.ValueOf(arr)
	if arrV.Kind() == reflect.Slice {
		for i := 0; i < arrV.Len(); i++ {
			// XXX - panics if slice element points to an unexported struct field
			// see https://golang.org/pkg/reflect/#Value.Interface
			if arrV.Index(i).Interface() == elem {
				return true
			}
		}
	}
	return false
}

func AddNumSuffix(i int) string {
	switch i % 10 {
	case 1:
		return fmt.Sprintf("%dst", i)
	case 2:
		return fmt.Sprintf("%dnd", i)
	case 3:
		return fmt.Sprintf("%drd", i)
	default:
		return fmt.Sprintf("%dth", i)
	}
}

const (
	numSecondsInYear = 31449600
	numSecondsInDay  = 86400
)

func UnixTimeToYearDay(s string) (int, error) {
	numSecondsSince1970, err := strconv.Atoi(s)
	if err != nil {
		return 0, commonerrors.ErrCannotParse
	}
	numYearsSince1970 := numSecondsSince1970 / numSecondsInYear
	year := 1970 + numSecondsSince1970
	isLeapYear := year != 2000 && year%4 == 0
	numLeapYearsSince1970 := numYearsSince1970 / 4

	yearday := numSecondsSince1970/numSecondsInDay - (numYearsSince1970)*365 + numLeapYearsSince1970
	if isLeapYear {
		yearday += 1
	}
	return yearday, nil
}
