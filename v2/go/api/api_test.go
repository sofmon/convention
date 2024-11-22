package api_test

import (
	"testing"
	"time"

	convAPI "github.com/sofmon/convention/v2/go/api"
	convAuth "github.com/sofmon/convention/v2/go/auth"
	convCtx "github.com/sofmon/convention/v2/go/ctx"
)

func Test_server_and_client(t *testing.T) {

	agentCtx := convCtx.New(convAuth.Claims{
		User: "Test_server_and_client",
	})

	go func() {
		err := convAPI.ListenAndServe(agentCtx, "localhost", 12345, authConfig, &APIImpl)
		if err != nil {
			t.Errorf("ListenAndServe() = %v; want nil", err)
		}
	}()

	time.Sleep(10 * time.Millisecond) // give time for the agent api to start

	client := convAPI.NewClient[API]("localhost", 12345)

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
}
