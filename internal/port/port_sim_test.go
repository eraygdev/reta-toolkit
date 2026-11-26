package port

import "testing"

func TestWindowsNetstatAndTasklistSimulation(t *testing.T) {
	tasklistCSV := `"System Idle Process","0","Services","0","8 K"` + "\r\n" +
		`"node.exe","4120","Console","1","45,120 K"` + "\r\n" +
		`"POSTGRES.EXE","2210","Services","0","22,400 K"` + "\r\n"

	taskMap := parseWindowsTasklistCSV(tasklistCSV)
	if taskMap["4120"] != "node" || taskMap["2210"] != "POSTGRES" {
		t.Fatalf("unexpected taskMap: %v", taskMap)
	}

	netstatOut := `
Active Connections

  Proto  Local Address          Foreign Address        State           PID
  TCP    0.0.0.0:135            0.0.0.0:0              LISTENING       888
  TCP    127.0.0.1:3000         0.0.0.0:0              LISTENING       4120
  TCP    [::1]:5432             [::]:0                 LISTENING       2210
  TCP    127.0.0.1:8080         0.0.0.0:0              LISTENING       9001
  TCP    127.0.0.1:3000         127.0.0.1:51234        ESTABLISHED     4120
  UDP    0.0.0.0:5353           *:*                                    1200
`
	dkrMap := map[string]string{"8080": "api-container"}
	entries := parseWindowsNetstat(netstatOut, taskMap, dkrMap)
	if len(entries) != 4 {
		t.Fatalf("expected 4 listening entries, got %+v", entries)
	}
	if entries[1] != (portEntry{Port: "3000", Command: "node", PID: "4120", Addr: "127.0.0.1:3000"}) {
		t.Fatalf("unexpected node entry: %+v", entries[1])
	}
	if entries[2] != (portEntry{Port: "5432", Command: "POSTGRES", PID: "2210", Addr: "[::1]:5432"}) {
		t.Fatalf("unexpected postgres entry: %+v", entries[2])
	}
	if entries[3] != (portEntry{Port: "8080", Command: "api-container", PID: "docker", Addr: "127.0.0.1:8080"}) {
		t.Fatalf("unexpected docker entry: %+v", entries[3])
	}
}

func TestMacOSLsofSimulation(t *testing.T) {
	lsofOut := `COMMAND     PID USER   FD   TYPE             DEVICE SIZE/OFF NODE NAME
ControlCe   456 eray    9u  IPv4 0x123456789abcdef0      0t0  TCP *:7000 (LISTEN)
node      12345 eray   22u  IPv6 0x123456789abcdef1      0t0  TCP [::1]:3000 (LISTEN)
python3   22222 eray    5u  IPv4 0x123456789abcdef2      0t0  TCP 127.0.0.1:8000 (LISTEN)
curl      33333 eray    6u  IPv4 0x123456789abcdef3      0t0  TCP 127.0.0.1:51000->127.0.0.1:8000 (ESTABLISHED)
`
	dkrMap := map[string]string{"8000": "web"}
	entries := parseLsofListen(lsofOut, dkrMap)
	if len(entries) != 3 {
		t.Fatalf("expected 3 listening entries, got %+v", entries)
	}
	if entries[0] != (portEntry{Port: "7000", Command: "ControlCe", PID: "456", Addr: "*:7000"}) {
		t.Fatalf("unexpected ControlCe entry: %+v", entries[0])
	}
	if entries[1] != (portEntry{Port: "3000", Command: "node", PID: "12345", Addr: "[::1]:3000"}) {
		t.Fatalf("unexpected IPv6 node entry: %+v", entries[1])
	}
	if entries[2] != (portEntry{Port: "8000", Command: "web", PID: "docker", Addr: "127.0.0.1:8000"}) {
		t.Fatalf("unexpected docker entry: %+v", entries[2])
	}
}

func TestLinuxSSSimulation(t *testing.T) {
	ssOut := `Netid State  Recv-Q Send-Q Local Address:Port Peer Address:Port Process
tcp   LISTEN 0      128    127.0.0.1:5432     0.0.0.0:*
tcp   LISTEN 0      128    0.0.0.0:8000       0.0.0.0:*         users:(("uvicorn",pid=991,fd=3))
udp   UNCONN 0      0      0.0.0.0:5353       0.0.0.0:*         users:(("LISTEN-daemon",pid=777,fd=4))
`
	entries := parseSSListen(ssOut, map[string]string{})
	if len(entries) != 2 {
		t.Fatalf("expected 2 LISTEN entries (ignoring UNCONN LISTEN-daemon), got %+v", entries)
	}
	if entries[0] != (portEntry{Port: "5432", Command: "postgres", PID: "-", Addr: "127.0.0.1:5432"}) {
		t.Fatalf("unexpected fallback service entry: %+v", entries[0])
	}
	if entries[1] != (portEntry{Port: "8000", Command: "uvicorn", PID: "991", Addr: "0.0.0.0:8000"}) {
		t.Fatalf("unexpected uvicorn entry: %+v", entries[1])
	}
}
