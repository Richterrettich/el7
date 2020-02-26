package etc

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"
)

type SingleLineCommentableConfigFile struct {
	Lines []*Line
}

func (s *SingleLineCommentableConfigFile) GetLineByPrefix(prefix string) *Line {
	for _, line := range s.Lines {
		if line.Type == EmptyLineType {
			continue
		}
		if strings.HasPrefix(line.Value, prefix) {
			return line
		}
	}
	return nil
}

func readSingleLineCommentableconfigFile(location string, commentPrefix string, valueCallback func(*Line) error) (*SingleLineCommentableConfigFile, error) {
	f, err := os.Open(location)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := new(SingleLineCommentableConfigFile)

	scanner := bufio.NewScanner(f)

	index := 0
	var line *Line
	for scanner.Scan() {
		lineContent := strings.TrimSpace(scanner.Text())
		line = &Line{
			Value:      lineContent,
			LineNumber: index,
		}

		switch {
		case len(lineContent) == 0:
			line.Type = EmptyLineType
		case strings.HasPrefix(lineContent, commentPrefix):
			line.Type = CommentLineType
		default:
			line.Type = ValueLineType
			err = valueCallback(line)
			if errors.Is(err, errNoMoreProcessing) {
				return result, nil
			}
			if err != nil {
				return nil, fmt.Errorf("error while processing file %s: %v", location, err)
			}
		}
		index++
	}

	if scanner.Err() != nil {
		return nil, fmt.Errorf("error while reading file %s: %v", location, err)
	}
	return result, nil
}

var errNoMoreProcessing error = fmt.Errorf("no more please")

type LineType uint16

const (
	EmptyLineType LineType = iota
	CommentLineType
	ValueLineType
)

type Line struct {
	Type       LineType
	Value      string
	LineNumber int
}

type stringValue struct {
	line  *Line
	value string
}

type ChronyConf struct {
	SingleLineCommentableConfigFile
	logDir *stringValue `conf "logdir"`
}

var chronyConfRegex regexp.Regexp = regexp.MustCompile("")

func ReadChronyConf(location string) (*ChronyConf, error) {

	result := new(ChronyConf)
	readSingleLineCommentableconfigFile(location, "#", func(l *Line) error {
		strings.Split(l.Value, " ")
		//setValueByTag(result, "conf", key string, newValue interface{})
		return nil
	})
}

func (c *ChronyConf) GetLogDirValue() string {
	if c.logDir != nil {
		return c.logDir.value
	}
	c.GetLineByPrefix("")
}

func setValueByTag(entity interface{}, tag, key string, newValue interface{}) error {
	val := reflect.ValueOf(entity)
	setVal := reflect.ValueOf(newValue)

	if val.Kind() != reflect.Ptr {
		return errors.Errorf("entity must be a pointer")
	}

	entityType := reflect.TypeOf(entity).Elem()

	s := val.Elem()

	if s.Kind() != reflect.Struct {
		return errors.Errorf("entity must be a struct")
	}

	for i := 0; i < s.NumField(); i++ {
		typeField := entityType.Field(i)
		property := typeField.Tag.Get(tag)
		if property == key {
			f := s.FieldByIndex([]int{i})
			if f.Kind() != setVal.Kind() {
				return errors.Errorf("datatypes not matching")
			}
			f.Set(setVal)
			return nil
		}
	}

	return errors.Errorf("no field with tag %s and name %s found", tag, key)
}

func getFieldByTag(entity interface{}, key string, tag string) (reflect.Value, iniTag) {
	entityType, val := getTypeAndVal(entity)
	for i := 0; i < val.NumField(); i++ {
		typeField := entityType.Field(i)
		property := typeField.Tag.Get(tag)
		if property == "" {
			continue
		}

		if property == key {
			return val.FieldByIndex([]int{i}), parsedTag
		}
	}
	panic(fmt.Errorf("no field with tag ini and name %s found", key))
}

func getTypeAndVal(entity interface{}) (reflect.Type, reflect.Value) {
	entityType := reflect.TypeOf(entity)
	val := reflect.ValueOf(entity)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
		entityType = entityType.Elem()
	}

	return entityType, val
}
