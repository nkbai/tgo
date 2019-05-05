package tracee

import (
	"debug/dwarf"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/nkbai/tgo/log"
)

const maxContainerItemsToPrint = 8

type value interface {
	String() string
	Size() int64
}

type int8Value struct {
	*dwarf.IntType
	val int8
}

func (v int8Value) String() string {
	return fmt.Sprintf("%d", v.val)
}

type int16Value struct {
	*dwarf.IntType
	val int16
}

func (v int16Value) String() string {
	return fmt.Sprintf("%d", v.val)
}

type int32Value struct {
	*dwarf.IntType
	val int32
}

func (v int32Value) String() string {
	return fmt.Sprintf("%d", v.val)
}

type int64Value struct {
	*dwarf.IntType
	val int64
}

func (v int64Value) String() string {
	return fmt.Sprintf("%d", v.val)
}

type uint8Value struct {
	*dwarf.UintType
	val uint8
}

func (v uint8Value) String() string {
	return fmt.Sprintf("%d", v.val)
}

type uint16Value struct {
	*dwarf.UintType
	val uint16
}

func (v uint16Value) String() string {
	return fmt.Sprintf("%d", v.val)
}

type uint32Value struct {
	*dwarf.UintType
	val uint32
}

func (v uint32Value) String() string {
	return fmt.Sprintf("%d", v.val)
}

type uint64Value struct {
	*dwarf.UintType
	val uint64
}

func (v uint64Value) String() string {
	return fmt.Sprintf("%d", v.val)
}

type float32Value struct {
	*dwarf.FloatType
	val float32
}

func (v float32Value) String() string {
	return fmt.Sprintf("%g", v.val)
}

type float64Value struct {
	*dwarf.FloatType
	val float64
}

func (v float64Value) String() string {
	return fmt.Sprintf("%g", v.val)
}

type complex64Value struct {
	*dwarf.ComplexType
	val complex64
}

func (v complex64Value) String() string {
	return fmt.Sprintf("%g", v.val)
}

type complex128Value struct {
	*dwarf.ComplexType
	val complex128
}

func (v complex128Value) String() string {
	return fmt.Sprintf("%g", v.val)
}

type boolValue struct {
	*dwarf.BoolType
	val bool
}

func (v boolValue) String() string {
	return fmt.Sprintf("%t", v.val)
}

type ptrValue struct {
	*dwarf.PtrType
	addr       uint64
	pointedVal value
}

func (v ptrValue) String() string {
	if v.pointedVal != nil {
		return fmt.Sprintf("&%s", v.pointedVal)
	}
	return fmt.Sprintf("%#x", v.addr)
}

type funcValue struct {
	*dwarf.FuncType
	addr uint64
}

func (v funcValue) String() string {
	return fmt.Sprintf("%#x", v.addr)
}

type stringValue struct {
	*dwarf.StructType
	val string
}

func (v stringValue) String() string {
	return strconv.Quote(v.val)
}

type sliceValue struct {
	*dwarf.StructType
	val []value
}

func (v sliceValue) String() string {
	if len(v.val) == 0 {
		return "nil"
	}

	var vals []string
	abbrev := false
	for i, v := range v.val {
		if i >= maxContainerItemsToPrint {
			abbrev = true
			break
		}
		vals = append(vals, v.String())
	}

	if abbrev {
		return fmt.Sprintf("[]{%s, ...}", strings.Join(vals, ", "))
	}
	return fmt.Sprintf("[]{%s}", strings.Join(vals, ", "))
}

type structValue struct {
	*dwarf.StructType
	fields      map[string]value
	abbreviated bool
}

func (v structValue) String() string {
	if v.abbreviated {
		return "{...}"
	}
	var vals []string
	for name, val := range v.fields {
		vals = append(vals, fmt.Sprintf("%s: %s", name, val))
	}
	return fmt.Sprintf("{%s}", strings.Join(vals, ", "))
}

type interfaceValue struct {
	*dwarf.StructType
	implType    dwarf.Type
	implVal     value
	abbreviated bool
}

func (v interfaceValue) String() string {
	if v.abbreviated {
		return "{...}"
	}
	if v.implType == nil {
		return "nil"
	}

	typeName := v.implType.String()
	const structPrefix = "struct "
	if strings.HasPrefix(typeName, structPrefix) {
		// just to make the logs cleaner
		typeName = strings.TrimPrefix(typeName, structPrefix)
	}
	return fmt.Sprintf("%s(%s)", typeName, v.implVal)
}

type arrayValue struct {
	*dwarf.ArrayType
	val []value
}

func (v arrayValue) String() string {
	var vals []string
	abbrev := false
	for i, v := range v.val {
		if i >= maxContainerItemsToPrint {
			abbrev = true
			break
		}
		vals = append(vals, v.String())
	}

	if abbrev {
		return fmt.Sprintf("[%d]{%s, ...}", len(vals), strings.Join(vals, ", "))
	}
	return fmt.Sprintf("[%d]{%s}", len(vals), strings.Join(vals, ", "))
}

type mapValue struct {
	*dwarf.TypedefType
	val map[value]value
}

func (v mapValue) String() string {
	var vals []string
	for k, v := range v.val {
		vals = append(vals, fmt.Sprintf("%s: %s", k, v))
	}
	return fmt.Sprintf("{%s}", strings.Join(vals, ", "))
}

type voidValue struct {
	dwarf.Type
	val []byte
}

func (v voidValue) String() string {
	return fmt.Sprintf("%v", v.val)
}

type valueParser struct {
	reader         memoryReader
	mapRuntimeType func(addr uint64) (dwarf.Type, error)
}

type memoryReader interface {
	ReadMemory(addr uint64, out []byte) error
}

// parseValue parses the `value` using the specified `rawTyp`.
// `remainingDepth` is the depth of parsing, and parser stops when the depth becomes negative.
// It is decremented when the struct type value is parsed, though the structs used by builtin types, such as slice and map, are not considered.
func (b valueParser) parseValue(rawTyp dwarf.Type, val []byte, remainingDepth int) value {
	switch typ := rawTyp.(type) {
	case *dwarf.IntType:
		switch typ.Size() {
		case 1:
			return int8Value{IntType: typ, val: int8(val[0])}
		case 2:
			return int16Value{IntType: typ, val: int16(binary.LittleEndian.Uint16(val))}
		case 4:
			return int32Value{IntType: typ, val: int32(binary.LittleEndian.Uint32(val))}
		case 8:
			return int64Value{IntType: typ, val: int64(binary.LittleEndian.Uint64(val))}
		}

	case *dwarf.UintType:
		switch typ.Size() {
		case 1:
			return uint8Value{UintType: typ, val: val[0]}
		case 2:
			return uint16Value{UintType: typ, val: binary.LittleEndian.Uint16(val)}
		case 4:
			return uint32Value{UintType: typ, val: binary.LittleEndian.Uint32(val)}
		case 8:
			return uint64Value{UintType: typ, val: binary.LittleEndian.Uint64(val)}
		}

	case *dwarf.FloatType:
		switch typ.Size() {
		case 4:
			return float32Value{FloatType: typ, val: math.Float32frombits(binary.LittleEndian.Uint32(val))}
		case 8:
			return float64Value{FloatType: typ, val: math.Float64frombits(binary.LittleEndian.Uint64(val))}
		}

	case *dwarf.ComplexType:
		switch typ.Size() {
		case 8:
			real := math.Float32frombits(binary.LittleEndian.Uint32(val[0:4]))
			img := math.Float32frombits(binary.LittleEndian.Uint32(val[4:8]))
			return complex64Value{ComplexType: typ, val: complex(real, img)}
		case 16:
			real := math.Float64frombits(binary.LittleEndian.Uint64(val[0:8]))
			img := math.Float64frombits(binary.LittleEndian.Uint64(val[8:16]))
			return complex128Value{ComplexType: typ, val: complex(real, img)}
		}

	case *dwarf.BoolType:
		return boolValue{BoolType: typ, val: val[0] == 1}

	case *dwarf.PtrType:
		addr := binary.LittleEndian.Uint64(val)
		if addr == 0 {
			// nil pointer
			return ptrValue{PtrType: typ}
		}

		if _, ok := typ.Type.(*dwarf.VoidType); ok {
			// unsafe.Pointer
			return ptrValue{PtrType: typ, addr: addr}
		}

		buff := make([]byte, typ.Type.Size())
		if err := b.reader.ReadMemory(addr, buff); err != nil {
			log.Debugf("failed to read memory (addr: %x): %v", addr, err)
			// the value may not be initialized yet (or too large)
			return ptrValue{PtrType: typ, addr: addr}
		}
		pointedVal := b.parseValue(typ.Type, buff, remainingDepth)
		return ptrValue{PtrType: typ, addr: addr, pointedVal: pointedVal}

	case *dwarf.FuncType:
		// TODO: print the pointer to the actual function (and the variables in closure if possible).
		addr := binary.LittleEndian.Uint64(val)
		return funcValue{FuncType: typ, addr: addr}

	case *dwarf.StructType:
		switch {
		case typ.StructName == "string":
			return b.parseStringValue(typ, val)
		case strings.HasPrefix(typ.StructName, "[]"):
			return b.parseSliceValue(typ, val, remainingDepth)
		case typ.StructName == "runtime.iface":
			return b.parseInterfaceValue(typ, val, remainingDepth)
		case typ.StructName == "runtime.eface":
			return b.parseEmptyInterfaceValue(typ, val, remainingDepth)
		default:
			return b.parseStructValue(typ, val, remainingDepth)
		}
	case *dwarf.ArrayType:
		if typ.Count == -1 {
			break
		}
		var vals []value
		stride := int(typ.Type.Size())
		for i := 0; i < int(typ.Count); i++ {
			vals = append(vals, b.parseValue(typ.Type, val[i*stride:(i+1)*stride], remainingDepth))
		}
		return arrayValue{ArrayType: typ, val: vals}
	case *dwarf.TypedefType:
		//if strings.HasPrefix(typ.String(), "map[") {
		//	return b.parseMapValue(typ, val, remainingDepth)
		//}

		// In this case, virtually do nothing so far. So do not decrement `remainingDepth`.
		return b.parseValue(typ.Type, val, remainingDepth)
	}
	return voidValue{Type: rawTyp, val: val}
}

func (b valueParser) parseStringValue(typ *dwarf.StructType, val []byte) stringValue {
	addr := binary.LittleEndian.Uint64(val[:8])
	len := int(binary.LittleEndian.Uint64(val[8:]))
	buff := make([]byte, len)

	if err := b.reader.ReadMemory(addr, buff); err != nil {
		log.Debugf("failed to read memory (addr: %x): %v", addr, err)
		return stringValue{StructType: typ}
	}
	return stringValue{StructType: typ, val: string(buff)}
}

func (b valueParser) parseSliceValue(typ *dwarf.StructType, val []byte, remainingDepth int) sliceValue {
	// Values are wrapped by slice struct. So +1 here.
	structVal := b.parseStructValue(typ, val, remainingDepth+1)
	length := int(structVal.fields["len"].(int64Value).val)
	if length == 0 {
		return sliceValue{StructType: typ}
	}

	firstElem := structVal.fields["array"].(ptrValue)
	sliceVal := sliceValue{StructType: typ, val: []value{firstElem.pointedVal}}

	for i := 1; i < length; i++ {
		addr := firstElem.addr + uint64(firstElem.pointedVal.Size())*uint64(i)
		buff := make([]byte, 8)
		binary.LittleEndian.PutUint64(buff, addr)
		elem := b.parseValue(firstElem.PtrType, buff, remainingDepth).(ptrValue)
		sliceVal.val = append(sliceVal.val, elem.pointedVal)
	}

	return sliceVal
}

func (b valueParser) parseInterfaceValue(typ *dwarf.StructType, val []byte, remainingDepth int) interfaceValue {
	// Interface is represented by the iface and itab struct. So remainingDepth needs to be at least 2.
	structVal := b.parseStructValue(typ, val, 2)
	ptrToTab := structVal.fields["tab"].(ptrValue)
	if ptrToTab.pointedVal == nil {
		return interfaceValue{StructType: typ}
	}
	if b.mapRuntimeType == nil {
		// Old go versions offer the different method to map the runtime type.
		return interfaceValue{StructType: typ, abbreviated: true}
	}

	tab := ptrToTab.pointedVal.(structValue)
	runtimeTypeAddr := tab.fields["_type"].(ptrValue).addr
	implType, err := b.mapRuntimeType(runtimeTypeAddr)
	if err != nil {
		log.Debugf("failed to find the impl type (runtime type addr: %x): %v", runtimeTypeAddr, err)
		return interfaceValue{StructType: typ}
	}

	data := structVal.fields["data"].(ptrValue)
	if _, ok := implType.(*dwarf.PtrType); ok {
		buff := make([]byte, 8)
		binary.LittleEndian.PutUint64(buff, data.addr)
		return interfaceValue{StructType: typ, implType: implType, implVal: b.parseValue(implType, buff, remainingDepth)}
	}

	// When the actual type is not pointer, we need the explicit dereference because data.addr is the pointer to the data.
	dataBuff := make([]byte, implType.Size())
	if err := b.reader.ReadMemory(data.addr, dataBuff); err != nil {
		log.Debugf("failed to read memory (addr: %x): %v", data.addr, err)
		return interfaceValue{StructType: typ}
	}
	return interfaceValue{StructType: typ, implType: implType, implVal: b.parseValue(implType, dataBuff, remainingDepth)}
}

func (b valueParser) parseEmptyInterfaceValue(typ *dwarf.StructType, val []byte, remainingDepth int) interfaceValue {
	// Empty interface is represented by the eface struct. So remainingDepth needs to be at least 1.
	structVal := b.parseStructValue(typ, val, 1)
	data := structVal.fields["data"].(ptrValue)
	if data.addr == 0 {
		return interfaceValue{StructType: typ}
	}
	if b.mapRuntimeType == nil {
		// Old go versions offer the different method to map the runtime type.
		return interfaceValue{StructType: typ, abbreviated: true}
	}

	runtimeTypeAddr := structVal.fields["_type"].(ptrValue).addr
	implType, err := b.mapRuntimeType(runtimeTypeAddr)
	if err != nil {
		log.Debugf("failed to find the impl type (runtime type addr: %x): %v", runtimeTypeAddr, err)
		return interfaceValue{StructType: typ}
	}

	if _, ok := implType.(*dwarf.PtrType); ok {
		buff := make([]byte, 8)
		binary.LittleEndian.PutUint64(buff, data.addr)
		return interfaceValue{StructType: typ, implType: implType, implVal: b.parseValue(implType, buff, remainingDepth)}
	}

	// When the actual type is not pointer, we need the explicit dereference because data.addr is the pointer to the data.
	dataBuff := make([]byte, implType.Size())
	if err := b.reader.ReadMemory(data.addr, dataBuff); err != nil {
		log.Debugf("failed to read memory (addr: %x): %v", data.addr, err)
		return interfaceValue{StructType: typ}
	}

	return interfaceValue{StructType: typ, implType: implType, implVal: b.parseValue(implType, dataBuff, remainingDepth)}
}

func (b valueParser) parseStructValue(typ *dwarf.StructType, val []byte, remainingDepth int) structValue {
	if remainingDepth <= 0 {
		return structValue{StructType: typ, abbreviated: true}
	}

	fields := make(map[string]value)
	for _, field := range typ.Field {
		fields[field.Name] = b.parseValue(field.Type, val[field.ByteOffset:field.ByteOffset+field.Type.Size()], remainingDepth-1)
	}
	return structValue{StructType: typ, fields: fields}
}

func (b valueParser) parseMapValue(typ *dwarf.TypedefType, val []byte, remainingDepth int) mapValue {
	// Actual keys and values are wrapped by hmap struct and buckets struct. So +2 here.
	ptrVal := b.parseValue(typ.Type, val, remainingDepth+2)
	if ptrVal.(ptrValue).pointedVal == nil {
		return mapValue{TypedefType: typ, val: nil}
	}

	hmapVal := ptrVal.(ptrValue).pointedVal.(structValue)
	numBuckets := 1 << hmapVal.fields["B"].(uint8Value).val
	ptrToBuckets := hmapVal.fields["buckets"].(ptrValue)
	ptrToOldBuckets := hmapVal.fields["oldbuckets"].(ptrValue)
	if ptrToOldBuckets.addr != 0 {
		log.Debugf("Map values may be defective")
	}

	mapValues := make(map[value]value)
	for i := 0; ; i++ {
		mapValuesInBucket := b.parseBucket(ptrToBuckets, remainingDepth)
		for k, v := range mapValuesInBucket {
			mapValues[k] = v
		}
		if i+1 == numBuckets {
			break
		}

		buckets := ptrToBuckets.pointedVal.(structValue)
		nextBucketAddr := ptrToBuckets.addr + uint64(buckets.Size())
		buff := make([]byte, 8)
		binary.LittleEndian.PutUint64(buff, nextBucketAddr)
		// Actual keys and values are wrapped by struct buckets. So +1 here.
		ptrToBuckets = b.parseValue(ptrToBuckets.PtrType, buff, remainingDepth+1).(ptrValue)
	}

	return mapValue{TypedefType: typ, val: mapValues}
}

func (b valueParser) parseBucket(ptrToBucket ptrValue, remainingDepth int) map[value]value {
	if ptrToBucket.addr == 0 {
		return nil // initialized map may not have bucket
	}

	mapValues := make(map[value]value)
	buckets := ptrToBucket.pointedVal.(structValue)
	tophash := buckets.fields["tophash"].(arrayValue)
	keys := buckets.fields["keys"].(arrayValue)
	values := buckets.fields["values"].(arrayValue)

	for j, hash := range tophash.val {
		if hash.(uint8Value).val == 0 {
			continue
		}
		mapValues[keys.val[j]] = values.val[j]
	}

	overflow := buckets.fields["overflow"].(ptrValue)
	if overflow.addr == 0 {
		return mapValues
	}

	buff := make([]byte, 8)
	binary.LittleEndian.PutUint64(buff, overflow.addr)
	// Actual keys and values are wrapped by struct buckets. So +1 here.
	ptrToOverflowBucket := b.parseValue(ptrToBucket.PtrType, buff, remainingDepth+1).(ptrValue)
	overflowedValues := b.parseBucket(ptrToOverflowBucket, remainingDepth)
	for k, v := range overflowedValues {
		mapValues[k] = v
	}
	return mapValues
}
