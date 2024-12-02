/*
Copyright 2024 The HAMi Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ascend

import (
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/Project-HAMi/HAMi/pkg/api"
	"github.com/Project-HAMi/HAMi/pkg/util"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

func Test_InitDevices(t *testing.T) {
	tests := []struct {
		name string
		args []VNPUConfig
		want []*Devices
	}{
		{
			name: "test with vaild configuration",
			args: []VNPUConfig{
				{
					ChipName:           "910B",
					CommonWord:         "Ascend910A",
					ResourceName:       "huawei.com/Ascend910A",
					ResourceMemoryName: "huawei.com/Ascend910A-memory",
					MemoryAllocatable:  int64(32768),
					MemoryCapacity:     int64(32768),
					AICore:             int32(30),
					Templates: []Template{
						{
							Name:   "vir02",
							Memory: int64(2184),
							AICore: int32(2),
						}, {
							Name:   "vir04",
							Memory: int64(4369),
							AICore: int32(4),
						}, {
							Name:   "vir08",
							Memory: int64(8738),
							AICore: int32(8),
						}, {
							Name:   "vir16",
							Memory: int64(17476),
							AICore: int32(16),
						},
					},
				},
			},
			want: []*Devices{
				{
					config: VNPUConfig{
						ChipName:           "910B",
						CommonWord:         "Ascend910A",
						ResourceName:       "huawei.com/Ascend910A",
						ResourceMemoryName: "huawei.com/Ascend910A-memory",
						MemoryAllocatable:  int64(32768),
						MemoryCapacity:     int64(32768),
						AICore:             int32(30),
						Templates: []Template{
							{
								Name:   "vir02",
								Memory: int64(2184),
								AICore: int32(2),
							}, {
								Name:   "vir04",
								Memory: int64(4369),
								AICore: int32(4),
							}, {
								Name:   "vir08",
								Memory: int64(8738),
								AICore: int32(8),
							}, {
								Name:   "vir16",
								Memory: int64(17476),
								AICore: int32(16),
							},
						},
					},
					nodeRegisterAnno: "hami.io/node-register-Ascend910A",
					useUUIDAnno:      "hami.io/use-Ascend910A-uuid",
					noUseUUIDAnno:    "hami.io/no-use-Ascend910A-uuid",
					handshakeAnno:    "hami.io/node-handshake-Ascend910A",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			enableAscend = true
			devices := InitDevices(test.args)
			if devices == nil {
				t.Error("Expected NvidiaGPUDevices to be initialized")
			}
			assert.Equal(t, len(devices), len(test.want), "Expected length of result to match want")
			for k, v := range devices {
				assert.Equal(t, v, devices[k], "load ascend vnpu config %s: %v", devices[k].config.CommonWord, devices[k].config)
			}
			assert.Equal(t, "hami.io/Ascend910A-devices-to-allocate", util.InRequestDevices[test.args[0].CommonWord])
			assert.Equal(t, "hami.io/Ascend910A-devices-allocated", util.SupportDevices[test.args[0].CommonWord])
			assert.Equal(t, test.want[0].handshakeAnno, util.HandshakeAnnos[test.args[0].CommonWord])
		})
	}
}

func Test_GetNodeDevices(t *testing.T) {
	dev := Devices{}
	tests := []struct {
		name string
		args corev1.Node
		want []*api.DeviceInfo
		err  error
	}{
		{
			name: "exist device",
			args: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-01",
					Annotations: map[string]string{
						dev.nodeRegisterAnno: "[{\"ID\":\"GPU-0\",\"Count\":4,\"Devmem\":8738,\"Devcore\":8,\"Type\":\"huawei.com/Ascend910\",\"Numa\":0,\"Health\":true}]",
					},
				},
			},
			want: []*api.DeviceInfo{
				{
					ID:      "GPU-0",
					Count:   int32(4),
					Devcore: int32(8),
					Devmem:  int32(8738),
					Type:    "huawei.com/Ascend910",
					Numa:    0,
					Health:  true,
				},
			},
			err: nil,
		},
		{
			name: "no device",
			args: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-02",
					Annotations: map[string]string{
						dev.nodeRegisterAnno: "[]",
					},
				},
			},
			want: []*api.DeviceInfo{},
			err:  errors.New("no device found on node"),
		},
		{
			name: "no annotation",
			args: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-03",
				},
			},
			want: []*api.DeviceInfo{},
			err:  fmt.Errorf("annos not found"),
		},
		{
			name: "failed to unmarshal node devices",
			args: corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-04",
					Annotations: map[string]string{
						dev.nodeRegisterAnno: "",
					},
				},
			},
			want: []*api.DeviceInfo{},
			err:  fmt.Errorf("failed to unmarshal node devices"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := dev.GetNodeDevices(test.args)
			if (err != nil) != (test.err != nil) {
				klog.ErrorS(err, "failed to unmarshal node devices", "node", test.args.Name)
			}
			if len(result) != len(test.want) {
				t.Errorf("GetNodeDevices got %d devices, want %d", len(result), len(test.want))
				return
			}
			if err == nil && len(result) != 0 {
				for k, v := range test.want {
					assert.Equal(t, v.Index, result[k].Index)
					assert.Equal(t, v.ID, result[k].ID)
					assert.Equal(t, v.Count, result[k].Count)
					assert.Equal(t, v.Devcore, result[k].Devcore)
					assert.Equal(t, v.Devmem, result[k].Devmem)
					assert.Equal(t, v.Type, result[k].Type)
					assert.Equal(t, v.Numa, result[k].Numa)
					assert.Equal(t, v.Health, result[k].Health)
				}
			}
		})
	}
}

func Test_PatchAnnotations(t *testing.T) {
	dev := Devices{
		config: VNPUConfig{
			CommonWord:     "Ascend910A",
			MemoryCapacity: int64(1024),
			Templates: []Template{
				{
					Name:   "vir02",
					Memory: int64(2184),
					AICore: int32(2),
				}, {
					Name:   "vir04",
					Memory: int64(4369),
					AICore: int32(4),
				}, {
					Name:   "vir08",
					Memory: int64(8738),
					AICore: int32(8),
				}, {
					Name:   "vir16",
					Memory: int64(17476),
					AICore: int32(16),
				},
			},
		},
	}
	tests := []struct {
		name string
		args struct {
			annoinput map[string]string
			pd        util.PodDevices
		}
		want map[string]string
	}{
		{
			name: "exist device",
			args: struct {
				annoinput map[string]string
				pd        util.PodDevices
			}{
				annoinput: map[string]string{},
				pd: util.PodDevices{
					dev.config.CommonWord: util.PodSingleDevice{
						[]util.ContainerDevice{
							{
								Idx:       0,
								UUID:      "device-0",
								Type:      "Ascend",
								Usedcores: 1,
								Usedmem:   8738,
							},
						},
					},
				},
			},
			want: map[string]string{
				util.InRequestDevices[dev.config.CommonWord]: "device-0,Ascend,8738,1:;",
				util.SupportDevices[dev.config.CommonWord]:   "device-0,Ascend,8738,1:;",
				"predicate-time":        strconv.FormatInt(time.Now().Unix(), 10),
				"huawei.com/Ascend910A": "[{\"UUID\":\"device-0\",\"temp\":\"vir08\"}]",
			},
		},
		{
			name: "no device",
			args: struct {
				annoinput map[string]string
				pd        util.PodDevices
			}{
				annoinput: map[string]string{},
				pd:        util.PodDevices{},
			},
			want: map[string]string{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := dev.PatchAnnotations(&test.args.annoinput, test.args.pd)

			assert.Equal(t, len(test.want), len(result), "Expected length of result to match want")
			for k, v := range test.want {
				assert.Equal(t, v, result[k], "pod add annotation key [%s], values is [%s]", k, result[k])
			}
		})
	}
}

func Test_CheckType(t *testing.T) {
	dev := Devices{
		config: VNPUConfig{
			CommonWord: "Ascend910A",
		},
	}
	tests := []struct {
		name string
		args struct {
			annos map[string]string
			d     util.DeviceUsage
			n     util.ContainerDeviceRequest
		}
		want bool
	}{
		{
			name: "the same type",
			args: struct {
				annos map[string]string
				d     util.DeviceUsage
				n     util.ContainerDeviceRequest
			}{
				annos: map[string]string{},
				d:     util.DeviceUsage{},
				n: util.ContainerDeviceRequest{
					Type: "Ascend910A",
				},
			},
			want: true,
		},
		{
			name: "the different type",
			args: struct {
				annos map[string]string
				d     util.DeviceUsage
				n     util.ContainerDeviceRequest
			}{
				annos: map[string]string{},
				d:     util.DeviceUsage{},
				n: util.ContainerDeviceRequest{
					Type: "Ascend910B",
				},
			},
			want: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, result, _ := dev.CheckType(test.args.annos, test.args.d, test.args.n)
			assert.Equal(t, result, test.want)
		})
	}
}

func Test_CheckUUID(t *testing.T) {
	dev := Devices{
		useUUIDAnno:   "hami.io/use-Ascend910A-uuid",
		noUseUUIDAnno: "hami.io/no-use-Ascend910A-uuid",
	}
	tests := []struct {
		name string
		args struct {
			annos map[string]string
			d     util.DeviceUsage
		}
		want bool
	}{
		{
			name: "don't set GPUUseUUID,GPUNoUseUUID and annotation",
			args: struct {
				annos map[string]string
				d     util.DeviceUsage
			}{
				annos: map[string]string{},
				d:     util.DeviceUsage{},
			},
			want: true,
		},
		{
			name: "set GPUUseUUID,don't set GPUNoUseUUID,annotation and device match",
			args: struct {
				annos map[string]string
				d     util.DeviceUsage
			}{
				annos: map[string]string{
					dev.useUUIDAnno: "test123,111",
				},
				d: util.DeviceUsage{
					ID: "test123",
				},
			},
			want: true,
		},
		{
			name: "don't set GPUUseUUID, set GPUNoUseUUID,annotation and device match",
			args: struct {
				annos map[string]string
				d     util.DeviceUsage
			}{
				annos: map[string]string{
					dev.noUseUUIDAnno: "test123,222",
				},
				d: util.DeviceUsage{
					ID: "test123",
				},
			},
			want: false,
		},
		{
			name: "set GPUUseUUID, don't set GPUNoUseUUID,annotation and device not match",
			args: struct {
				annos map[string]string
				d     util.DeviceUsage
			}{
				annos: map[string]string{
					dev.useUUIDAnno: "test123,222",
				},
				d: util.DeviceUsage{
					ID: "test456",
				},
			},
			want: false,
		},
		{
			name: "don't set GPUUseUUID, set GPUNoUseUUID,annotation and device not match",
			args: struct {
				annos map[string]string
				d     util.DeviceUsage
			}{
				annos: map[string]string{
					dev.noUseUUIDAnno: "test123,222",
				},
				d: util.DeviceUsage{
					ID: "test456",
				},
			},
			want: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := dev.CheckUUID(test.args.annos, test.args.d)
			assert.Equal(t, result, test.want)
		})
	}
}

func Test_CheckHealth(t *testing.T) {
	dev := Devices{}
	tests := []struct {
		name string
		args struct {
			devType string
			n       corev1.Node
		}
		want1 bool
		want2 bool
	}{
		{
			name: "Requesting state",
			args: struct {
				devType string
				n       corev1.Node
			}{
				devType: "huawei.com/Ascend910",
				n: corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							util.HandshakeAnnos["huawei.com/Ascend910"]: "Requesting_2128.12.02 00:00:00",
						},
					},
				},
			},
			want1: true,
			want2: false,
		},
		{
			name: "Deleted state",
			args: struct {
				devType string
				n       corev1.Node
			}{
				devType: "huawei.com/Ascend910",
				n: corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							util.HandshakeAnnos["huawei.com/Ascend910"]: "Deleted",
						},
					},
				},
			},
			want1: true,
			want2: false,
		},
		{
			name: "Unknown state",
			args: struct {
				devType string
				n       corev1.Node
			}{
				devType: "huawei.com/Ascend910",
				n: corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							util.HandshakeAnnos["huawei.com/Ascend910"]: "Unknown",
						},
					},
				},
			},
			want1: true,
			want2: true,
		},
		{
			name: "Requesting state expired",
			args: struct {
				devType string
				n       corev1.Node
			}{
				devType: "huawei.com/Ascend910",
				n: corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							util.HandshakeAnnos["huawei.com/Ascend910"]: "Requesting_2024.01.02 00:00:00",
						},
					},
				},
			},
			want1: false,
			want2: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result1, result2 := dev.CheckHealth(test.args.devType, &test.args.n)
			assert.Equal(t, result1, test.want1)
			assert.Equal(t, result2, test.want2)
		})
	}
}
