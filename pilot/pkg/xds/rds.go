// Copyright Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package xds

import (
	"time"

	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pilot/pkg/networking/util"
	v3 "istio.io/istio/pilot/pkg/xds/v3"
	"istio.io/istio/pkg/util/protomarshal"
)

func (s *DiscoveryServer) pushRoute(con *Connection, push *model.PushContext, version string) error {
	pushStart := time.Now()
	rawRoutes := s.ConfigGenerator.BuildHTTPRoutes(con.proxy, push, con.Routes())
	if s.DebugConfigs {
		for _, r := range rawRoutes {
			con.XdsRoutes[r.Name] = r
			if adsLog.DebugEnabled() {
				resp, _ := protomarshal.ToJSONWithIndent(r, " ")
				adsLog.Debugf("RDS: Adding route:%s for node:%v", resp, con.proxy.ID)
			}
		}
	}

	response := routeDiscoveryResponse(rawRoutes, version, push.Version)
	err := con.send(response)
	rdsPushTime.Record(time.Since(pushStart).Seconds())
	if err != nil {
		recordSendError("RDS", con.ConID, rdsSendErrPushes, err)
		return err
	}
	rdsPushes.Increment()

	adsLog.Infof("RDS: PUSH for node:%s routes:%d", con.proxy.ID, len(rawRoutes))
	return nil
}

func routeDiscoveryResponse(rs []*route.RouteConfiguration, version, noncePrefix string) *discovery.DiscoveryResponse {
	resp := &discovery.DiscoveryResponse{
		TypeUrl:     v3.RouteType,
		VersionInfo: version,
		Nonce:       nonce(noncePrefix),
	}
	for _, rc := range rs {
		resp.Resources = append(resp.Resources, util.MessageToAny(rc))
	}

	return resp
}
