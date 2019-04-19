package fpnn

type Payload struct {
	data map[interface{}]interface{}
}

func NewPayload() *Payload {
	payload := &Payload{}
	payload.data = make(map[interface{}]interface{})
	return payload
}

func convertToInt64(value interface{}, unconvertPanic bool) int64 {
	switch value.(type) {
	case int64:
		return int64(value.(int64))
	case int32:
		return int64(value.(int32))
	case int16:
		return int64(value.(int16))
	case int8:
		return int64(value.(int8))
	case int:
		return int64(value.(int))

	case uint64:
		return int64(value.(uint64))
	case uint32:
		return int64(value.(uint32))
	case uint16:
		return int64(value.(uint16))
	case uint8:
		return int64(value.(uint8))
	case uint:
		return int64(value.(uint))

	case float32:
		return int64(value.(float32))
	case float64:
		return int64(value.(float64))

	default:
		if !unconvertPanic {
			return 0
		} else {
			panic("Type convert failed.")
		}
	}
}

func convertToUint64(value interface{}, unconvertPanic bool) uint64 {
	switch value.(type) {
	case int64:
		return uint64(value.(int64))
	case int32:
		return uint64(value.(int32))
	case int16:
		return uint64(value.(int16))
	case int8:
		return uint64(value.(int8))
	case int:
		return uint64(value.(int))

	case uint64:
		return uint64(value.(uint64))
	case uint32:
		return uint64(value.(uint32))
	case uint16:
		return uint64(value.(uint16))
	case uint8:
		return uint64(value.(uint8))
	case uint:
		return uint64(value.(uint))

	case float32:
		return uint64(value.(float32))
	case float64:
		return uint64(value.(float64))

	default:
		if !unconvertPanic {
			return 0
		} else {
			panic("Type convert failed.")
		}
	}
}

func convertToFloat64(value interface{}, unconvertPanic bool) float64 {
	switch value.(type) {
	case int64:
		return float64(value.(int64))
	case int32:
		return float64(value.(int32))
	case int16:
		return float64(value.(int16))
	case int8:
		return float64(value.(int8))
	case int:
		return float64(value.(int))

	case uint64:
		return float64(value.(uint64))
	case uint32:
		return float64(value.(uint32))
	case uint16:
		return float64(value.(uint16))
	case uint8:
		return float64(value.(uint8))
	case uint:
		return float64(value.(uint))

	case float32:
		return float64(value.(float32))
	case float64:
		return float64(value.(float64))

	default:
		if !unconvertPanic {
			return 0
		} else {
			panic("Type convert failed.")
		}
	}
}

func convertToString(value interface{}, unconvertPanic bool) string {
	switch value.(type) {
	case string:
		return value.(string)
	case []byte:
		return string(value.([]byte))
	case []rune:
		return string(value.([]rune))
	default:
		if !unconvertPanic {
			return ""
		} else {
			panic("Type convert failed.")
		}
	}
}

//---------------------[ Get Methods ]----------------------------//

func (payload *Payload) GetInt64(key string) (int64, bool) {
	val, ok := payload.data[key]
	return convertToInt64(val, false), ok
}
func (payload *Payload) GetInt32(key string) (int32, bool) {
	val, ok := payload.data[key]
	return int32(convertToInt64(val, false)), ok
}
func (payload *Payload) GetInt16(key string) (int16, bool) {
	val, ok := payload.data[key]
	return int16(convertToInt64(val, false)), ok
}
func (payload *Payload) GetInt8(key string) (int8, bool) {
	val, ok := payload.data[key]
	return int8(convertToInt64(val, false)), ok
}
func (payload *Payload) GetInt(key string) (int, bool) {
	val, ok := payload.data[key]
	return int(convertToInt64(val, false)), ok
}

func (payload *Payload) GetUint64(key string) (uint64, bool) {
	val, ok := payload.data[key]
	return convertToUint64(val, false), ok
}
func (payload *Payload) GetUint32(key string) (uint32, bool) {
	val, ok := payload.data[key]
	return uint32(convertToUint64(val, false)), ok
}
func (payload *Payload) GetUint16(key string) (uint16, bool) {
	val, ok := payload.data[key]
	return uint16(convertToUint64(val, false)), ok
}
func (payload *Payload) GetUint8(key string) (uint8, bool) {
	val, ok := payload.data[key]
	return uint8(convertToUint64(val, false)), ok
}
func (payload *Payload) GetUint(key string) (uint, bool) {
	val, ok := payload.data[key]
	return uint(convertToUint64(val, false)), ok
}

func (payload *Payload) GetString(key string) (value string, ok bool) {
	val, ok := payload.data[key]

	defer func() {
		recover();
	}()

	value = convertToString(val, false)
	return value, ok
}

func (payload *Payload) GetBool(key string) (value bool, ok bool) {
	val, ok := payload.data[key]

	defer func() {
		recover();
	}()

	value = val.(bool)
	return value, ok
}
func (payload *Payload) GetFloat32(key string) (float32, bool) {
	val, ok := payload.data[key]
	return float32(convertToFloat64(val, false)), ok
}
func (payload *Payload) GetFloat64(key string) (float64, bool) {
	val, ok := payload.data[key]
	return convertToFloat64(val, false), ok
}

func (payload *Payload) GetSlice(key string) (value []interface{}, ok bool) {
	val, ok := payload.data[key]

	defer func() {
		recover();
	}()

	value = val.([]interface{})

	return value, ok
}

func (payload *Payload) GetMap(key string) (value map[interface{}]interface{}, ok bool) {
	val, ok := payload.data[key]

	defer func() {
		recover();
	}()

	value = val.(map[interface{}]interface{})

	return value, ok
}

func (payload *Payload) GetDict(key string) (value *Payload, ok bool) {
	val, ok := payload.data[key]
	if !ok {
		return nil, ok
	}

	defer func() {
		recover();
	}()

	value = &Payload{}
	value.data = val.(map[interface{}]interface{})
	return value, ok
}

//---------------------[ Want Methods ]----------------------------//

func (payload *Payload) WantInt64(key string) (int64, bool) {
	val, ok := payload.data[key]
	return convertToInt64(val, true), ok
}
func (payload *Payload) WantInt32(key string) (int32, bool) {
	val, ok := payload.data[key]
	return int32(convertToInt64(val, true)), ok
}
func (payload *Payload) WantInt16(key string) (int16, bool) {
	val, ok := payload.data[key]
	return int16(convertToInt64(val, true)), ok
}
func (payload *Payload) WantInt8(key string) (int8, bool) {
	val, ok := payload.data[key]
	return int8(convertToInt64(val, true)), ok
}
func (payload *Payload) WantInt(key string) (int, bool) {
	val, ok := payload.data[key]
	return int(convertToInt64(val, true)), ok
}

func (payload *Payload) WantUint64(key string) (uint64, bool) {
	val, ok := payload.data[key]
	return convertToUint64(val, true), ok
}
func (payload *Payload) WantUint32(key string) (uint32, bool) {
	val, ok := payload.data[key]
	return uint32(convertToUint64(val, true)), ok
}
func (payload *Payload) WantUint16(key string) (uint16, bool) {
	val, ok := payload.data[key]
	return uint16(convertToUint64(val, true)), ok
}
func (payload *Payload) WantUint8(key string) (uint8, bool) {
	val, ok := payload.data[key]
	return uint8(convertToUint64(val, true)), ok
}
func (payload *Payload) WantUint(key string) (uint, bool) {
	val, ok := payload.data[key]
	return uint(convertToUint64(val, true)), ok
}

func (payload *Payload) WantString(key string) (string, bool) {
	value, ok := payload.data[key]
	return convertToString(value, true), ok
}

func (payload *Payload) WantBool(key string) (bool, bool) {
	value, ok := payload.data[key]
	return value.(bool), ok
}

func (payload *Payload) WantFloat32(key string) (float32, bool) {
	val, ok := payload.data[key]
	return float32(convertToFloat64(val, true)), ok
}
func (payload *Payload) WantFloat64(key string) (float64, bool) {
	val, ok := payload.data[key]
	return convertToFloat64(val, true), ok
}

func (payload *Payload) WantSlice(key string) ([]interface{}, bool) {
	value, ok := payload.data[key]
	return value.([]interface{}), ok
}

func (payload *Payload) WantMap(key string) (map[interface{}]interface{}, bool) {
	value, ok := payload.data[key]
	return value.(map[interface{}]interface{}), ok
}

func (payload *Payload) WantDict(key string) (*Payload, bool) {
	value, ok := payload.data[key]
	if !ok {
		return nil, ok
	}

	newDict := &Payload{}
	newDict.data = value.(map[interface{}]interface{})
	return newDict, ok
}

//---------------------[ Other Method ]----------------------------//

func (payload *Payload) Param(key string, value interface{}) {
	switch value.(type) {
	default:
		payload.data[key] = value
	case Payload:
		var tmp Payload = value.(Payload)
		payload.data[key] = tmp.data
	case *Payload:
		var tmp *Payload = value.(*Payload)
		payload.data[key] = tmp.data
	}
}

func (payload *Payload) Get(key string) (value interface{}, ok bool) {
	value, ok = payload.data[key]
	return value, ok
}

func (payload *Payload) Exist(key string) bool {
	_, ok := payload.data[key]
	return ok 
}
