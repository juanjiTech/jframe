package settings

import (
	"context"
	"encoding"
	"encoding/json"
	"errors"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

var mock = false

func MockInit() {
	mock = true
}

func MockReset() {
	mock = false
}

var ItemMap = make(map[string]UpdateableItem)

type UpdateableItem interface {
	ValueBytes() []byte
	UpdateValueBytes([]byte) error
}

type Item[T any] struct {
	Label        string `json:"label"`
	Key          string `json:"key"`
	group        string
	ItemOrder    int `json:"order"`
	CurrValue    T   `json:"value"`
	DefaultValue T   `json:"defaultValue"`

	afterUpdate []func()
}

func (o *Item[T]) Init() {
	o.Value()
}

func (o *Item[T]) Order() int {
	return o.ItemOrder
}

func (o *Item[T]) UpdateValueBytes(v []byte) error {
	dst := new(T)
	err := json.Unmarshal(v, dst)
	if err != nil {
		return err
	}
	return o.Update(*dst)
}

func (o *Item[T]) Update(v T) error {
	vb, _ := json.Marshal(v)
	settingCache.Set(o.Key, vb, cache.NoExpiration)
	o.CurrValue = v
	if mock || store == nil {
		o.triggerAfterUpdateHook()
		return nil
	}

	err := store.UpdateOrCreate(context.Background(), &Record{
		Key:   o.Key,
		Value: vb,
		Order: o.ItemOrder,
	})
	if err != nil {
		return err
	}
	o.triggerAfterUpdateHook()
	return nil
}

func (o *Item[T]) value(data []byte) (T, error) {
	err := json.Unmarshal(data, &o.CurrValue)
	if err != nil {
		o.CurrValue = o.DefaultValue
		return o.CurrValue, err
	}
	return o.CurrValue, nil
}

func (o *Item[T]) ValueBytes() []byte {
	v, ok := settingCache.Get(o.Key)
	if ok {
		return v.([]byte)
	}
	if mock || store == nil {
		vb, _ := json.Marshal(o.DefaultValue)
		settingCache.Set(o.Key, vb, cache.NoExpiration)
		return vb
	}

	record, err := store.GetByKey(context.Background(), o.Key)
	if err != nil {
		vb, _ := json.Marshal(o.DefaultValue)
		if errors.Is(err, ErrNotFound) {
			settingCache.Set(o.Key, vb, cache.NoExpiration)
		}
		return vb
	}

	settingCache.Set(o.Key, record.Value, cache.NoExpiration)
	return record.Value
}

func (o *Item[T]) Value() T {
	value, _ := o.value(o.ValueBytes())
	return value
}

func (o *Item[T]) AfterUpdateHook(f func()) {
	o.afterUpdate = append(o.afterUpdate, f)
}

func (o *Item[T]) triggerAfterUpdateHook() {
	var wg sync.WaitGroup
	for _, f := range o.afterUpdate {
		wg.Add(1)
		go func(f func()) {
			defer func() {
				if err := recover(); err != nil {
					zap.S().Errorw("trigger after update hook failed", "error", err)
				}
			}()
			defer wg.Done()
			f()
		}(f)
	}
	wg.Wait()
}

func NewItem[T any](key, label string, defaultValue T) *Item[T] {
	if key == "" {
		panic("empty key")
	}
	defer func() { currGroup.currChildrenOrder++ }()
	return NewItemWithGroupAndOrder(key, label, currGroup.Path, currGroup.currChildrenOrder, defaultValue)
}

func GetDefaultValue[T any](key string, defaultValue T) T {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}

	var result any
	var err error

	switch any(defaultValue).(type) {
	case string:
		result = value
	case bool:
		result, err = strconv.ParseBool(value)
	case int:
		var tmp int64
		tmp, err = strconv.ParseInt(value, 10, 0)
		result = int(tmp)
	case int8:
		var tmp int64
		tmp, err = strconv.ParseInt(value, 10, 8)
		result = int8(tmp)
	case int16:
		var tmp int64
		tmp, err = strconv.ParseInt(value, 10, 16)
		result = int16(tmp)
	case int32:
		var tmp int64
		tmp, err = strconv.ParseInt(value, 10, 32)
		result = int32(tmp)
	case int64:
		result, err = strconv.ParseInt(value, 10, 64)
	case uint:
		var tmp uint64
		tmp, err = strconv.ParseUint(value, 10, 0)
		result = uint(tmp)
	case uint8:
		var tmp uint64
		tmp, err = strconv.ParseUint(value, 10, 8)
		result = uint8(tmp)
	case uint16:
		var tmp uint64
		tmp, err = strconv.ParseUint(value, 10, 16)
		result = uint16(tmp)
	case uint32:
		var tmp uint64
		tmp, err = strconv.ParseUint(value, 10, 32)
		result = uint32(tmp)
	case uint64:
		result, err = strconv.ParseUint(value, 10, 64)
	case float32:
		var tmp float64
		tmp, err = strconv.ParseFloat(value, 32)
		result = float32(tmp)
	case float64:
		result, err = strconv.ParseFloat(value, 64)
	default:
		dst := any(&defaultValue)
		if unmarshaler, ok := dst.(encoding.TextUnmarshaler); ok {
			err = unmarshaler.UnmarshalText([]byte(value))
			if err != nil {
				return defaultValue
			}
			return defaultValue
		}
		return defaultValue
	}

	if err != nil {
		return defaultValue
	}

	return result.(T)
}

func NewItemWithGroupAndOrder[T any](key, label, group string, order int, defaultValue T) *Item[T] {
	envKey := group + "/" + key
	envKey = strings.TrimPrefix(envKey, "root/")
	envKey = strings.ToUpper(strings.ReplaceAll(envKey, "/", "_"))
	defaultValue = GetDefaultValue(envKey, defaultValue)

	item := &Item[T]{
		Label:        label,
		Key:          group + ":" + key,
		group:        group,
		ItemOrder:    order,
		DefaultValue: defaultValue,
	}
	currGroup.Children = append(currGroup.Children, item)
	if _, ok := ItemMap[item.Key]; ok {
		panic("setting item key conflict")
	}
	ItemMap[item.Key] = item
	return item
}

var (
	rootGroup = &GroupInfo{Path: "root", currChildrenOrder: 1}
	currGroup = rootGroup
)

func Root() *GroupInfo {
	return rootGroup
}

type GroupInfo struct {
	parent     *GroupInfo
	Path       string `json:"path"`
	Label      string `json:"label"`
	GroupOrder int    `json:"order"`

	currChildrenOrder int
	Children          []any `json:"children"`
}

func (o *GroupInfo) Init() {
	for _, child := range o.Children {
		if g, ok := child.(initAble); ok {
			g.Init()
		}
	}
}

func (o *GroupInfo) SortChildren() {
	sort.Slice(o.Children, func(i, j int) bool {
		return o.Children[i].(orderAble).Order() < o.Children[j].(orderAble).Order()
	})
	for _, child := range o.Children {
		if g, ok := child.(*GroupInfo); ok {
			g.SortChildren()
		}
	}
}

func (o *GroupInfo) Order() int {
	return o.GroupOrder
}

func Group(key, name string, f func()) {
	currGroup = &GroupInfo{
		parent:     currGroup,
		Path:       currGroup.Path + "/" + key,
		Label:      name,
		GroupOrder: currGroup.currChildrenOrder,
	}
	currGroup.parent.Children = append(currGroup.parent.Children, currGroup)
	f()
	currGroup = currGroup.parent
}

type initAble interface {
	Init()
}

type orderAble interface {
	Order() int
}
