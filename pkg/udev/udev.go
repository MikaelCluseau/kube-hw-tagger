package udev

// #cgo LDFLAGS: -ludev
// #include <libudev.h>
import "C"

import (
	"fmt"
	"strconv"
)

type DeviceEvent struct {
	Action string
	Device *Device
}

type Device struct {
	ParentSysName string
	DevPath       string
	Subsystem     string
	DevType       string
	SysPath       string
	SysName       string
	SysNum        string
	DevNode       string
	Driver        string
	Properties    map[string]string
	Tags          map[string]string
	SysAttrs      map[string]string
}

var udev *C.struct_udev

func init() {
	udev = New()
}

func MonitorDeviceEvents(subsystem string, events chan DeviceEvent) error {
	monitor := udev.NewMonitorFromNetlink("udev")
	if monitor == nil {
		return fmt.Errorf("error: unable to create netlink socket to udev")
	}
	defer func() {
		monitor.Close()
	}()

	monitor.AddMatchSubsystem(subsystem)
	if err := monitor.EnableReceiving(); err != nil {
		return err
	}

	for {
		device := monitor.ReceiveDevice()
		if device == nil {
			continue
		}
		events <- device.Event()
		device.Close()
	}
}

func SubsystemDevices(subsystem string) []*Device {
	enum := udev.NewEnumerate()
	enum.AddMatchSubsystem(subsystem)
	enum.ScanDevices()
	devices := make([]*Device, 0)
	for entry := enum.ListFront(); entry != nil; entry = entry.Next() {
		path := entry.Name()
		dev := udev.NewDeviceFromSyspath(path)
		devices = append(devices, dev.Device())
		dev.Close()
	}
	enum.Close()
	return devices
}

// udev
func New() *C.struct_udev {
	return C.udev_new()
}
func (udev *C.struct_udev) Close() {
	C.udev_unref(udev)
}

// udev enumerate
func (udev *C.struct_udev) NewEnumerate() *C.struct_udev_enumerate {
	return C.udev_enumerate_new(udev)
}
func (enum *C.struct_udev_enumerate) AddMatchSubsystem(subsystem string) {
	C.udev_enumerate_add_match_subsystem(enum, C.CString(subsystem))
}
func (enum *C.struct_udev_enumerate) ScanDevices() {
	C.udev_enumerate_scan_devices(enum)
}
func (enum *C.struct_udev_enumerate) ListFront() *C.struct_udev_list_entry {
	return C.udev_enumerate_get_list_entry(enum)
}
func (enum *C.struct_udev_enumerate) Close() {
	C.udev_enumerate_unref(enum)
}

// udev device
func (udev *C.struct_udev) NewDeviceFromSyspath(syspath string) *C.struct_udev_device {
	return C.udev_device_new_from_syspath(udev, C.CString(syspath))
}
func (dev *C.struct_udev_device) Action() string {
	return C.GoString(C.udev_device_get_action(dev))
}
func (dev *C.struct_udev_device) DevPath() string {
	return C.GoString(C.udev_device_get_devpath(dev))
}
func (dev *C.struct_udev_device) Subsystem() string {
	return C.GoString(C.udev_device_get_subsystem(dev))
}
func (dev *C.struct_udev_device) DevType() string {
	return C.GoString(C.udev_device_get_devtype(dev))
}
func (dev *C.struct_udev_device) SysPath() string {
	return C.GoString(C.udev_device_get_syspath(dev))
}
func (dev *C.struct_udev_device) SysName() string {
	return C.GoString(C.udev_device_get_sysname(dev))
}
func (dev *C.struct_udev_device) SysNum() string {
	return C.GoString(C.udev_device_get_sysnum(dev))
}
func (dev *C.struct_udev_device) DevNode() string {
	return C.GoString(C.udev_device_get_devnode(dev))
}
func (dev *C.struct_udev_device) Driver() string {
	return C.GoString(C.udev_device_get_driver(dev))
}
func (dev *C.struct_udev_device) Close() {
	C.udev_device_unref(dev)
}

// - properties
func (dev *C.struct_udev_device) PropertyValue(sysattr string) string {
	return C.GoString(C.udev_device_get_property_value(dev, C.CString(sysattr)))
}
func (dev *C.struct_udev_device) PropertiesListFront() *C.struct_udev_list_entry {
	return C.udev_device_get_properties_list_entry(dev)
}

// - sysattrs
func (dev *C.struct_udev_device) SysattrValue(sysattr string) string {
	return C.GoString(C.udev_device_get_sysattr_value(dev, C.CString(sysattr)))
}
func (dev *C.struct_udev_device) SysattrListFront() *C.struct_udev_list_entry {
	return C.udev_device_get_sysattr_list_entry(dev)
}

// - tags
func (dev *C.struct_udev_device) TagsListFront() *C.struct_udev_list_entry {
	return C.udev_device_get_tags_list_entry(dev)
}

// - parents
func (dev *C.struct_udev_device) Parent() *C.struct_udev_device {
	return C.udev_device_get_parent(dev)
}
func (dev *C.struct_udev_device) ParentWithSubsystemDevtype(subsystem, devtype string) *C.struct_udev_device {
	return C.udev_device_get_parent_with_subsystem_devtype(dev, C.CString(subsystem), C.CString(devtype))
}

// - device
func (dev *C.struct_udev_device) Device() *Device {
	device := Device{
		DevPath:    dev.DevPath(),
		Subsystem:  dev.Subsystem(),
		DevType:    dev.DevType(),
		SysPath:    dev.SysPath(),
		SysName:    dev.SysName(),
		SysNum:     dev.SysNum(),
		DevNode:    dev.DevNode(),
		Driver:     dev.Driver(),
		Properties: make(map[string]string),
		Tags:       make(map[string]string),
		SysAttrs:   make(map[string]string),
	}
	parent := dev.Parent()
	if parent != nil {
		device.ParentSysName = parent.SysName()
	}
	for entry := dev.TagsListFront(); entry != nil; entry = entry.Next() {
		device.Tags[entry.Name()] = entry.Value()
	}
	for entry := dev.PropertiesListFront(); entry != nil; entry = entry.Next() {
		device.Properties[entry.Name()] = dev.PropertyValue(entry.Name())
	}
	for entry := dev.SysattrListFront(); entry != nil; entry = entry.Next() {
		device.SysAttrs[entry.Name()] = dev.SysattrValue(entry.Name())
	}
	return &device
}
func (dev *C.struct_udev_device) Event() DeviceEvent {
	return DeviceEvent{
		Action: dev.Action(),
		Device: dev.Device(),
	}
}

// Device
func (dev *Device) FsUuid() string {
	return dev.Properties["ID_FS_UUID"]
}

// udev monitor
func (udev *C.struct_udev) NewMonitorFromNetlink(link string) *C.struct_udev_monitor {
	return C.udev_monitor_new_from_netlink(udev, C.CString(link))
}

func (monitor *C.struct_udev_monitor) AddMatchSubsystem(subsystem string) {
	C.udev_monitor_filter_add_match_subsystem_devtype(monitor, C.CString(subsystem), nil)
}
func (monitor *C.struct_udev_monitor) EnableReceiving() error {
	if C.udev_monitor_enable_receiving(monitor) < 0 {
		return fmt.Errorf("error: unable to subscribe to udev events")
	}
	return nil
}
func (monitor *C.struct_udev_monitor) ReceiveDevice() *C.struct_udev_device {
	return C.udev_monitor_receive_device(monitor)
}
func (monitor *C.struct_udev_monitor) Close() {
	C.udev_monitor_unref(monitor)
}

// udev list entry
func (entry *C.struct_udev_list_entry) Next() *C.struct_udev_list_entry {
	return C.udev_list_entry_get_next(entry)
}
func (entry *C.struct_udev_list_entry) Name() string {
	return C.GoString(C.udev_list_entry_get_name(entry))
}
func (entry *C.struct_udev_list_entry) Value() string {
	return C.GoString(C.udev_list_entry_get_value(entry))
}

func atoi(value string) int {
	v, err := strconv.Atoi(value)
	if err != nil {
		panic(err)
	}
	return v
}
