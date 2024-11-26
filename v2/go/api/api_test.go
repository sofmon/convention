package api_test

import (
	"errors"
	"testing"
	"time"

	convAPI "github.com/sofmon/convention/v2/go/api"
	convAuth "github.com/sofmon/convention/v2/go/auth"
	convCtx "github.com/sofmon/convention/v2/go/ctx"
)

func Test_server_and_client(t *testing.T) {

	const (
		roleTrigger convAuth.Role = "trigger"
		roleIn      convAuth.Role = "in"
		roleOut     convAuth.Role = "out"
		roleInOut   convAuth.Role = "inout"

		permissionTrigger   convAuth.Permission = "trigger"
		permissionTriggerP1 convAuth.Permission = "trigger_p1"
		permissionTriggerP2 convAuth.Permission = "trigger_p2"
		permissionTriggerP3 convAuth.Permission = "trigger_p3"
		permissionTriggerP4 convAuth.Permission = "trigger_p4"
		permissionIn        convAuth.Permission = "in"
		permissionInP1      convAuth.Permission = "in_p1"
		permissionInP2      convAuth.Permission = "in_p2"
		permissionInP3      convAuth.Permission = "in_p3"
		permissionInP4      convAuth.Permission = "in_p4"
		permissionOut       convAuth.Permission = "out"
		permissionOutP1     convAuth.Permission = "out_p1"
		permissionOutP2     convAuth.Permission = "out_p2"
		permissionOutP3     convAuth.Permission = "out_p3"
		permissionOutP4     convAuth.Permission = "out_p4"
		permissionInOut     convAuth.Permission = "inout"
		permissionInOutP1   convAuth.Permission = "inout_p1"
		permissionInOutP2   convAuth.Permission = "inout_p2"
		permissionInOutP3   convAuth.Permission = "inout_p3"
		permissionInOutP4   convAuth.Permission = "inout_p4"
	)

	var authConfig = convAuth.Config{
		Roles: convAuth.RolePermissions{
			roleTrigger: convAuth.Permissions{
				permissionTrigger,
				permissionTriggerP1,
				permissionTriggerP2,
				permissionTriggerP3,
				permissionTriggerP4,
			},
			roleIn: convAuth.Permissions{
				permissionIn,
				permissionInP1,
				permissionInP2,
				permissionInP3,
				permissionInP4,
			},
			roleOut: convAuth.Permissions{
				permissionOut,
				permissionOutP1,
				permissionOutP2,
				permissionOutP3,
				permissionOutP4,
			},
			roleInOut: convAuth.Permissions{
				permissionInOut,
				permissionInOutP1,
				permissionInOutP2,
				permissionInOutP3,
				permissionInOutP4,
			},
		},
		Permissions: convAuth.PermissionActions{
			permissionTrigger: convAuth.Actions{
				"HEAD /test/v1/trigger",
			},
			permissionTriggerP1: convAuth.Actions{
				"HEAD /test/v1/trigger/p1/{any}",
			},
			permissionTriggerP2: convAuth.Actions{
				"HEAD /test/v1/trigger/p1/{any}/p2/{any}",
			},
			permissionTriggerP3: convAuth.Actions{
				"HEAD /test/v1/trigger/p1/{any}/p2/{any}/p3/{any}",
			},
			permissionTriggerP4: convAuth.Actions{
				"HEAD /test/v1/trigger/p1/{any}/p2/{any}/p3/{any}/p4/{any}",
			},
			permissionIn: convAuth.Actions{
				"PUT /test/v1/in",
			},
			permissionInP1: convAuth.Actions{
				"PUT /test/v1/in/p1/{any}",
			},
			permissionInP2: convAuth.Actions{
				"PUT /test/v1/in/p1/{any}/p2/{any}",
			},
			permissionInP3: convAuth.Actions{
				"PUT /test/v1/in/p1/{any}/p2/{any}/p3/{any}",
			},
			permissionInP4: convAuth.Actions{
				"PUT /test/v1/in/p1/{any}/p2/{any}/p3/{any}/p4/{any}",
			},
			permissionOut: convAuth.Actions{
				"GET /test/v1/out",
			},
			permissionOutP1: convAuth.Actions{
				"GET /test/v1/out/p1/{any}",
			},
			permissionOutP2: convAuth.Actions{
				"GET /test/v1/out/p1/{any}/p2/{any}",
			},
			permissionOutP3: convAuth.Actions{
				"GET /test/v1/out/p1/{any}/p2/{any}/p3/{any}",
			},
			permissionOutP4: convAuth.Actions{
				"GET /test/v1/out/p1/{any}/p2/{any}/p3/{any}/p4/{any}",
			},
			permissionInOut: convAuth.Actions{
				"POST /test/v1/inout",
			},
			permissionInOutP1: convAuth.Actions{
				"POST /test/v1/inout/p1/{any}",
			},
			permissionInOutP2: convAuth.Actions{
				"POST /test/v1/inout/p1/{any}/p2/{any}",
			},
			permissionInOutP3: convAuth.Actions{
				"POST /test/v1/inout/p1/{any}/p2/{any}/p3/{any}",
			},
			permissionInOutP4: convAuth.Actions{
				"POST /test/v1/inout/p1/{any}/p2/{any}/p3/{any}/p4/{any}",
			},
		},
		Public: convAuth.Actions{
			"GET /test/v1/openapi.yaml",
		},
	}

	type P1 string
	type P2 string
	type P3 string
	type P4 string

	type InObj struct {
		P1 P1 `json:"p1"`
		P2 P2 `json:"p2"`
		P3 P3 `json:"p3"`
		P4 P4 `json:"p4"`
	}

	type OutObj struct {
		P1 P1 `json:"p1"`
		P2 P2 `json:"p2"`
		P3 P3 `json:"p3"`
		P4 P4 `json:"p4"`
	}

	type API struct {
		GetOpenAPI convAPI.OpenAPI `api:"GET /test/v1/openapi.yaml"`

		Trigger   convAPI.Trigger                   `api:"HEAD /test/v1/trigger"`
		TriggerP1 convAPI.TriggerP1[P1]             `api:"HEAD /test/v1/trigger/p1/{p1}"`
		TriggerP2 convAPI.TriggerP2[P1, P2]         `api:"HEAD /test/v1/trigger/p1/{p1}/p2/{p2}"`
		TriggerP3 convAPI.TriggerP3[P1, P2, P3]     `api:"HEAD /test/v1/trigger/p1/{p1}/p2/{p2}/p3/{p3}"`
		TriggerP4 convAPI.TriggerP4[P1, P2, P3, P4] `api:"HEAD /test/v1/trigger/p1/{p1}/p2/{p2}/p3/{p3}/p4/{p4}"`

		In   convAPI.In[InObj]                   `api:"PUT /test/v1/in"`
		InP1 convAPI.InP1[InObj, P1]             `api:"PUT /test/v1/in/p1/{p1}"`
		InP2 convAPI.InP2[InObj, P1, P2]         `api:"PUT /test/v1/in/p1/{p1}/p2/{p2}"`
		InP3 convAPI.InP3[InObj, P1, P2, P3]     `api:"PUT /test/v1/in/p1/{p1}/p2/{p2}/p3/{p3}"`
		InP4 convAPI.InP4[InObj, P1, P2, P3, P4] `api:"PUT /test/v1/in/p1/{p1}/p2/{p2}/p3/{p3}/p4/{p4}"`

		Out   convAPI.Out[OutObj]                   `api:"GET /test/v1/out"`
		OutP1 convAPI.OutP1[OutObj, P1]             `api:"GET /test/v1/out/p1/{p1}"`
		OutP2 convAPI.OutP2[OutObj, P1, P2]         `api:"GET /test/v1/out/p1/{p1}/p2/{p2}"`
		OutP3 convAPI.OutP3[OutObj, P1, P2, P3]     `api:"GET /test/v1/out/p1/{p1}/p2/{p2}/p3/{p3}"`
		OutP4 convAPI.OutP4[OutObj, P1, P2, P3, P4] `api:"GET /test/v1/out/p1/{p1}/p2/{p2}/p3/{p3}/p4/{p4}"`

		InOut   convAPI.InOut[InObj, OutObj]                   `api:"POST /test/v1/inout"`
		InOutP1 convAPI.InOutP1[InObj, OutObj, P1]             `api:"POST /test/v1/inout/p1/{p1}"`
		InOutP2 convAPI.InOutP2[InObj, OutObj, P1, P2]         `api:"POST /test/v1/inout/p1/{p1}/p2/{p2}"`
		InOutP3 convAPI.InOutP3[InObj, OutObj, P1, P2, P3]     `api:"POST /test/v1/inout/p1/{p1}/p2/{p2}/p3/{p3}"`
		InOutP4 convAPI.InOutP4[InObj, OutObj, P1, P2, P3, P4] `api:"POST /test/v1/inout/p1/{p1}/p2/{p2}/p3/{p3}/p4/{p4}"`
	}

	inObjMatch := func(a InObj, p ...any) error {
		for _, p := range p {
			if p, ok := p.(P1); ok && a.P1 != p {
				return errors.New("p1 does not match")
			}
			if p, ok := p.(P2); ok && a.P2 != p {
				return errors.New("p2 does not match")
			}
			if p, ok := p.(P3); ok && a.P3 != p {
				return errors.New("p3 does not match")
			}
			if p, ok := p.(P4); ok && a.P4 != p {
				return errors.New("p4 does not match")
			}
		}
		return nil
	}

	var APIImpl = API{
		Trigger: convAPI.NewTrigger(func(ctx convCtx.Context) error {
			return nil
		}),
		TriggerP1: convAPI.NewTriggerP1(func(ctx convCtx.Context, p1 P1) error {
			return nil
		}),
		TriggerP2: convAPI.NewTriggerP2(func(ctx convCtx.Context, p1 P1, p2 P2) error {
			return nil
		}),
		TriggerP3: convAPI.NewTriggerP3(func(ctx convCtx.Context, p1 P1, p2 P2, p3 P3) error {
			return nil
		}),
		TriggerP4: convAPI.NewTriggerP4(func(ctx convCtx.Context, p1 P1, p2 P2, p3 P3, p4 P4) error {
			return nil
		}),

		In: convAPI.NewIn(func(ctx convCtx.Context, in InObj) error {
			return nil
		}),
		InP1: convAPI.NewInP1(func(ctx convCtx.Context, p1 P1, in InObj) error {
			return inObjMatch(in, p1)
		}),
		InP2: convAPI.NewInP2(func(ctx convCtx.Context, p1 P1, p2 P2, in InObj) error {
			return inObjMatch(in, p1, p2)
		}),
		InP3: convAPI.NewInP3(func(ctx convCtx.Context, p1 P1, p2 P2, p3 P3, in InObj) error {
			return inObjMatch(in, p1, p2, p3)
		}),
		InP4: convAPI.NewInP4(func(ctx convCtx.Context, p1 P1, p2 P2, p3 P3, p4 P4, in InObj) error {
			return inObjMatch(in, p1, p2, p3, p4)
		}),

		Out: convAPI.NewOut(func(ctx convCtx.Context) (OutObj, error) {
			return OutObj{}, nil
		}),
		OutP1: convAPI.NewOutP1(func(ctx convCtx.Context, p1 P1) (OutObj, error) {
			return OutObj{P1: p1}, nil
		}),
		OutP2: convAPI.NewOutP2(func(ctx convCtx.Context, p1 P1, p2 P2) (OutObj, error) {
			return OutObj{P1: p1, P2: p2}, nil
		}),
		OutP3: convAPI.NewOutP3(func(ctx convCtx.Context, p1 P1, p2 P2, p3 P3) (OutObj, error) {
			return OutObj{P1: p1, P2: p2, P3: p3}, nil
		}),
		OutP4: convAPI.NewOutP4(func(ctx convCtx.Context, p1 P1, p2 P2, p3 P3, p4 P4) (OutObj, error) {
			return OutObj{P1: p1, P2: p2, P3: p3, P4: p4}, nil
		}),

		InOut: convAPI.NewInOut(func(ctx convCtx.Context, in InObj) (OutObj, error) {
			return OutObj{}, nil
		}),
		InOutP1: convAPI.NewInOutP1(func(ctx convCtx.Context, p1 P1, in InObj) (OutObj, error) {
			return OutObj{P1: p1}, inObjMatch(in, p1)
		}),
		InOutP2: convAPI.NewInOutP2(func(ctx convCtx.Context, p1 P1, p2 P2, in InObj) (OutObj, error) {
			return OutObj{P1: p1, P2: p2}, inObjMatch(in, p1, p2)
		}),
		InOutP3: convAPI.NewInOutP3(func(ctx convCtx.Context, p1 P1, p2 P2, p3 P3, in InObj) (OutObj, error) {
			return OutObj{P1: p1, P2: p2, P3: p3}, inObjMatch(in, p1, p2, p3)
		}),
		InOutP4: convAPI.NewInOutP4(func(ctx convCtx.Context, p1 P1, p2 P2, p3 P3, p4 P4, in InObj) (OutObj, error) {
			return OutObj{P1: p1, P2: p2, P3: p3, P4: p4}, inObjMatch(in, p1, p2, p3, p4)
		}),
	}

	agentCtx := convCtx.New(convAuth.Claims{
		User: "Test_server_and_client",
	})

	svr, err := convAPI.NewServer(agentCtx, "localhost", portForAPITest(t), authConfig, &APIImpl)
	if err != nil {
		t.Errorf("ListenAndServe() = %v; want nil", err)
	}

	go svr.ListenAndServe()

	time.Sleep(10 * time.Millisecond) // give time for the agent api to start

	client := convAPI.NewClient[API]("localhost", portForAPITest(t))

	callerCtxRoleTrigger := convCtx.New(convAuth.Claims{User: "test-v1", Roles: convAuth.Roles{roleTrigger}})
	callerCtxRoleIn := convCtx.New(convAuth.Claims{User: "test-v1", Roles: convAuth.Roles{roleIn}})
	callerCtxRoleOut := convCtx.New(convAuth.Claims{User: "test-v1", Roles: convAuth.Roles{roleOut}})
	callerCtxRoleInOut := convCtx.New(convAuth.Claims{User: "test-v1", Roles: convAuth.Roles{roleInOut}})

	if err := client.Trigger.Call(callerCtxRoleTrigger); err != nil {
		t.Errorf("Trigger.Call() = %v; want nil", err)
	}
	if err := client.Trigger.Call(callerCtxRoleIn); err == nil {
		t.Errorf("Trigger.Call() = nil; want error")
	}
	if err := client.TriggerP1.Call(callerCtxRoleTrigger, "p1"); err != nil {
		t.Errorf("TriggerP1.Call() = %v; want nil", err)
	}
	if err := client.TriggerP1.Call(callerCtxRoleIn, "p1"); err == nil {
		t.Errorf("TriggerP1.Call() = nil; want error")
	}
	if err := client.TriggerP2.Call(callerCtxRoleTrigger, "p1", "p2"); err != nil {
		t.Errorf("TriggerP2.Call() = %v; want nil", err)
	}
	if err := client.TriggerP2.Call(callerCtxRoleIn, "p1", "p2"); err == nil {
		t.Errorf("TriggerP2.Call() = nil; want error")
	}
	if err := client.TriggerP3.Call(callerCtxRoleTrigger, "p1", "p2", "p3"); err != nil {
		t.Errorf("TriggerP3.Call() = %v; want nil", err)
	}
	if err := client.TriggerP3.Call(callerCtxRoleIn, "p1", "p2", "p3"); err == nil {
		t.Errorf("TriggerP3.Call() = nil; want error")
	}
	if err := client.TriggerP4.Call(callerCtxRoleTrigger, "p1", "p2", "p3", "p4"); err != nil {
		t.Errorf("TriggerP4.Call() = %v; want nil", err)
	}
	if err := client.TriggerP4.Call(callerCtxRoleIn, "p1", "p2", "p3", "p4"); err == nil {
		t.Errorf("TriggerP4.Call() = nil; want error")
	}

	if err := client.In.Call(callerCtxRoleIn, InObj{}); err != nil {
		t.Errorf("In.Call() = %v; want nil", err)
	}
	if err := client.In.Call(callerCtxRoleTrigger, InObj{}); err == nil {
		t.Errorf("In.Call() = nil; want error")
	}
	if err := client.InP1.Call(callerCtxRoleIn, "p1", InObj{P1: "p1"}); err != nil {
		t.Errorf("InP1.Call() = %v; want nil", err)
	}
	if err := client.InP1.Call(callerCtxRoleTrigger, "p1", InObj{P1: "p1"}); err == nil {
		t.Errorf("InP1.Call() = nil; want error")
	}
	if err := client.InP2.Call(callerCtxRoleIn, "p1", "p2", InObj{P1: "p1", P2: "p2"}); err != nil {
		t.Errorf("InP2.Call() = %v; want nil", err)
	}
	if err := client.InP2.Call(callerCtxRoleTrigger, "p1", "p2", InObj{P1: "p1", P2: "p2"}); err == nil {
		t.Errorf("InP2.Call() = nil; want error")
	}
	if err := client.InP3.Call(callerCtxRoleIn, "p1", "p2", "p3", InObj{P1: "p1", P2: "p2", P3: "p3"}); err != nil {
		t.Errorf("InP3.Call() = %v; want nil", err)
	}
	if err := client.InP3.Call(callerCtxRoleTrigger, "p1", "p2", "p3", InObj{P1: "p1", P2: "p2", P3: "p3"}); err == nil {
		t.Errorf("InP3.Call() = nil; want error")
	}
	if err := client.InP4.Call(callerCtxRoleIn, "p1", "p2", "p3", "p4", InObj{P1: "p1", P2: "p2", P3: "p3", P4: "p4"}); err != nil {
		t.Errorf("InP4.Call() = %v; want nil", err)
	}
	if err := client.InP4.Call(callerCtxRoleTrigger, "p1", "p2", "p3", "p4", InObj{P1: "p1", P2: "p2", P3: "p3", P4: "p4"}); err == nil {
		t.Errorf("InP4.Call() = nil; want error")
	}

	if _, err := client.Out.Call(callerCtxRoleOut); err != nil {
		t.Errorf("Out.Call() = %v; want nil", err)
	}
	if _, err := client.Out.Call(callerCtxRoleTrigger); err == nil {
		t.Errorf("Out.Call() = nil; want error")
	}
	if _, err := client.OutP1.Call(callerCtxRoleOut, "p1"); err != nil {
		t.Errorf("OutP1.Call() = %v; want nil", err)
	}
	if _, err := client.OutP1.Call(callerCtxRoleTrigger, "p1"); err == nil {
		t.Errorf("OutP1.Call() = nil; want error")
	}
	if _, err := client.OutP2.Call(callerCtxRoleOut, "p1", "p2"); err != nil {
		t.Errorf("OutP2.Call() = %v; want nil", err)
	}
	if _, err := client.OutP2.Call(callerCtxRoleTrigger, "p1", "p2"); err == nil {
		t.Errorf("OutP2.Call() = nil; want error")
	}
	if _, err := client.OutP3.Call(callerCtxRoleOut, "p1", "p2", "p3"); err != nil {
		t.Errorf("OutP3.Call() = %v; want nil", err)
	}
	if _, err := client.OutP3.Call(callerCtxRoleTrigger, "p1", "p2", "p3"); err == nil {
		t.Errorf("OutP3.Call() = nil; want error")
	}
	if _, err := client.OutP4.Call(callerCtxRoleOut, "p1", "p2", "p3", "p4"); err != nil {
		t.Errorf("OutP4.Call() = %v; want nil", err)
	}
	if _, err := client.OutP4.Call(callerCtxRoleTrigger, "p1", "p2", "p3", "p4"); err == nil {
		t.Errorf("OutP4.Call() = nil; want error")
	}

	if _, err := client.InOut.Call(callerCtxRoleInOut, InObj{}); err != nil {
		t.Errorf("InOut.Call() = %v; want nil", err)
	}
	if _, err := client.InOut.Call(callerCtxRoleTrigger, InObj{}); err == nil {
		t.Errorf("InOut.Call() = nil; want error")
	}
	if _, err := client.InOutP1.Call(callerCtxRoleInOut, "p1", InObj{P1: "p1"}); err != nil {
		t.Errorf("InOutP1.Call() = %v; want nil", err)
	}
	if _, err := client.InOutP1.Call(callerCtxRoleTrigger, "p1", InObj{P1: "p1"}); err == nil {
		t.Errorf("InOutP1.Call() = nil; want error")
	}
	if _, err := client.InOutP2.Call(callerCtxRoleInOut, "p1", "p2", InObj{P1: "p1", P2: "p2"}); err != nil {
		t.Errorf("InOutP2.Call() = %v; want nil", err)
	}
	if _, err := client.InOutP2.Call(callerCtxRoleTrigger, "p1", "p2", InObj{P1: "p1", P2: "p2"}); err == nil {
		t.Errorf("InOutP2.Call() = nil; want error")
	}
	if _, err := client.InOutP3.Call(callerCtxRoleInOut, "p1", "p2", "p3", InObj{P1: "p1", P2: "p2", P3: "p3"}); err != nil {
		t.Errorf("InOutP3.Call() = %v; want nil", err)
	}
	if _, err := client.InOutP3.Call(callerCtxRoleTrigger, "p1", "p2", "p3", InObj{P1: "p1", P2: "p2", P3: "p3"}); err == nil {
		t.Errorf("InOutP3.Call() = nil; want error")
	}
	if _, err := client.InOutP4.Call(callerCtxRoleInOut, "p1", "p2", "p3", "p4", InObj{P1: "p1", P2: "p2", P3: "p3", P4: "p4"}); err != nil {
		t.Errorf("InOutP4.Call() = %v; want nil", err)
	}
	if _, err := client.InOutP4.Call(callerCtxRoleTrigger, "p1", "p2", "p3", "p4", InObj{P1: "p1", P2: "p2", P3: "p3", P4: "p4"}); err == nil {
		t.Errorf("InOutP4.Call() = nil; want error")
	}

	err = svr.Shutdown(agentCtx)
	if err != nil {
		t.Errorf("Shutdown() = %v; want nil", err)
	}
}
