package motion

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/andreacioni/motionctrl/utils"

	"github.com/stretchr/testify/require"
)

func TestConfigParser(t *testing.T) {
	parseConfig("motion_test.conf")
}

func TestNotPresentParser(t *testing.T) {
	configMap, _ := parseConfig("motion_test.conf")

	value := configMap["this_is_not_present"]

	require.Equal(t, "", value)
}

func TestCheck(t *testing.T) {
	configMap, err := parseConfig("motion_test.conf")

	require.NoError(t, err)

	err = checkConfig(configMap)

	require.NoError(t, err)

	require.Equal(t, len(configReadOnlyParams)-2, len(configMap), "Configuration parameters map must contain %d elements", len(configReadOnlyParams))
}

func TestCheckInstall(t *testing.T) {
	err := checkInstall()

	require.NoError(t, err)
}

func TestStartStop(t *testing.T) {
	require.NoError(t, Init("motion_test.conf", false, false))

	require.NoError(t, Startup(false))

	started, err := IsStarted()

	require.NoError(t, err)
	require.True(t, started)

	require.NoError(t, Shutdown())
}

func TestIsRunning(t *testing.T) {

	err := ioutil.WriteFile("/tmp/motion.pid", []byte("1234567"), 0666)

	require.NoError(t, err)

	require.NoError(t, Init("motion_test.conf", false, false))

	require.NoError(t, Startup(false))

	started, err := IsStarted()

	require.NoError(t, err)
	require.True(t, started)

	require.NoError(t, Shutdown())
}

func TestStartStopAutostart(t *testing.T) {
	require.NoError(t, Init("motion_test.conf", true, false))

	started, err := IsStarted()

	require.NoError(t, err)
	require.True(t, started)

	require.NoError(t, Shutdown())
}

func TestRestart(t *testing.T) {

	require.NoError(t, Init("motion_test.conf", false, false))

	require.NoError(t, Startup(false))

	started, err := IsStarted()

	require.NoError(t, err)
	require.True(t, started)

	require.NoError(t, Restart())

	require.NoError(t, Shutdown())
}

func TestConfigTypeMapper(t *testing.T) {
	testMap := map[string]string{
		"value1": "text",
		"value2": "off",
		"value3": "on",
		"value4": "3",
	}

	require.Equal(t, "text", ConfigTypeMapper(testMap["value1"]))
	require.Equal(t, false, ConfigTypeMapper(testMap["value2"]))
	require.Equal(t, true, ConfigTypeMapper(testMap["value3"]))
	require.Equal(t, 3, ConfigTypeMapper(testMap["value4"]))
}

func TestRegexConfigFileParser(t *testing.T) {
	testString := "#comment here\n;comment here\nhello 12\nword 11\nnullparam (null)\nonoff on\noffon off"

	testMap := utils.RegexSubmatchTypedMap(configDefaultParserRegex, testString, ConfigTypeMapper)

	require.Equal(t, 5, len(testMap))
	require.Equal(t, 12, testMap["hello"])
	require.Equal(t, 11, testMap["word"])
	require.Empty(t, testMap["nullparam"])
	require.Equal(t, true, testMap["onoff"])
	require.Equal(t, false, testMap["offon"])
}

func TestRegexConfigList(t *testing.T) {
	testString := "#comment = here\n;comment = here\nhello = 12\nword = 11\nnullparam = (null)\nonoff = on\noffon = off"

	testMap := utils.RegexSubmatchTypedMap(listConfigParserRegex, testString, ConfigTypeMapper)

	require.Equal(t, 5, len(testMap))
	require.Equal(t, 12, testMap["hello"])
	require.Equal(t, 11, testMap["word"])
	require.Empty(t, testMap["nullparam"])
	require.Equal(t, true, testMap["onoff"])
	require.Equal(t, false, testMap["offon"])
}

func TestRegexSetRegex(t *testing.T) {
	testString := "testparam = Hello\nDone"
	testURL := "/config/set?daemon=true"

	require.True(t, utils.RegexMustMatch(fmt.Sprintf(setConfigParserRegex, "testparam", "Hello"), testString))

	mapped := utils.RegexSubmatchTypedMap("/config/set\\?("+KeyValueRegex+"+)=("+KeyValueRegex+"+)", testURL, nil)
	require.Equal(t, 1, len(mapped))

}

func TestParticularStartAndStop(t *testing.T) {
	require.NoError(t, Init("motion_test.conf", false, false))

	require.NoError(t, Startup(false))

	started, err := IsStarted()

	require.NoError(t, err)
	require.True(t, started)

	ret, err := ConfigGet("log_level") //Changing daemon instead of 'log_level' cause Shutdown to fail

	require.NoError(t, err)
	require.Equal(t, 6, ret.(int))

	err = ConfigSet("log_level", "5")

	require.NoError(t, err)

	ret, err = ConfigGet("log_level")

	require.NoError(t, err)
	require.Equal(t, 5, ret.(int))

	require.NoError(t, Shutdown())
}

func TestSomeConfigs(t *testing.T) {
	conf, _ := parseConfig("motion_test.conf")
	require.Equal(t, "/tmp", conf[ConfigTargetDir])
}

func TestWaitLiveRegex(t *testing.T) {
	text := "Motion 4.1.1+gitfcc66b8 Running [1] Camera\n0\n"

	require.True(t, utils.RegexMustMatch(waitLiveRegex, text))
}
