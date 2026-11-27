package api

import (
	"strings"
	"testing"
)

func TestSavedTicketEndpoints(t *testing.T) {
	a, match := newDataAPI(t)
	const ticketID = "0b6f3c2e-6a1d-4b8e-9f1a-2c3d4e5f6a7b"
	if res := call(t, a, "POST", "/api/tickets/analyse",
		`{"ticket_id":"`+ticketID+`","origin":"manual","legs":[{"match_id":"`+match+`","market":"HOME_WIN"}]}`, nil); res.StatusCode != 200 {
		t.Fatalf("analyse: %d", res.StatusCode)
	}

	var saved struct {
		ID     string
		Status string
		Legs   []struct {
			Original struct{ Market, Outcome string }
		}
	}
	res := call(t, a, "POST", "/api/saved", `{"ticket_id":"`+ticketID+`"}`, &saved)
	if res.StatusCode != 200 || len(saved.ID) != 22 || saved.Status != "pending" || len(saved.Legs) != 1 ||
		saved.Legs[0].Original.Market != "HOME_WIN" || saved.Legs[0].Original.Outcome != "pending" {
		t.Fatalf("save: %d %+v", res.StatusCode, saved)
	}
	if cc := res.Header.Get("Cache-Control"); !strings.Contains(cc, "no-store") {
		t.Fatalf("Cache-Control = %q", cc)
	}

	if res := call(t, a, "GET", "/api/saved/"+saved.ID, "", nil); res.StatusCode != 200 {
		t.Fatalf("open link: %d", res.StatusCode)
	}
	var list []map[string]any
	if call(t, a, "GET", "/api/saved?ids="+saved.ID+",AAAAAAAAAAAAAAAAAAAAAA", "", &list); len(list) != 1 || list[0]["legs"] != nil {
		t.Fatalf("list: %v", list)
	}
	var p problem
	if res := call(t, a, "GET", "/api/saved/nope", "", &p); res.StatusCode != 404 || !strings.Contains(p.Detail, "90 days") {
		t.Fatalf("unknown link: %d %q", res.StatusCode, p.Detail)
	}
	if res := call(t, a, "DELETE", "/api/saved/"+saved.ID, `{"ticket_id":"another-ticket-id"}`, nil); res.StatusCode != 404 {
		t.Fatalf("delete with the wrong ticket: %d", res.StatusCode)
	}
	if res := call(t, a, "DELETE", "/api/saved/"+saved.ID, `{"ticket_id":"`+ticketID+`"}`, nil); res.StatusCode != 204 {
		t.Fatalf("delete: %d", res.StatusCode)
	}
}
