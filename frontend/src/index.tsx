import React from "react";
import ReactDOM from "react-dom/client";
import useWebSocket, { ReadyState } from "react-use-websocket";
const WS_URL = "ws://127.0.0.1:8333/ws";

import Table from "./table";
import "./index.css";

import { createColumnHelper } from "@tanstack/react-table";
import Accordion from "react-bootstrap/Accordion";
import Dropdown from "react-bootstrap/Dropdown";
import Badge from "react-bootstrap/Badge";
import Stack from "react-bootstrap/Stack";
import Button from "react-bootstrap/Button";
import { Trash } from "react-bootstrap-icons";

import "bootstrap/dist/css/bootstrap.min.css";

function App() {
  const [socketUrl] = React.useState(WS_URL);
  const didUnmount = React.useRef(false);

  const { sendMessage, lastMessage, readyState } = useWebSocket(socketUrl, {
    shouldReconnect: () => {
      /*
        useWebSocket will handle unmounting for you, but this is an example of a 
        case in which you would not want it to automatically reconnect
      */
      return didUnmount.current === false;
    },
    reconnectAttempts: 10,
    reconnectInterval: 3000,
  });

  React.useEffect(() => {
    return () => {
      didUnmount.current = true;
    };
  }, []);

  React.useEffect(() => {
    console.log("Connection state changed");
    if (readyState === ReadyState.OPEN) {
      sendMessage(
        JSON.stringify({
          type: "subscribe",
          payload: { topics: ["rules", "agents"] },
        }),
      );
    }
  }, [readyState]);

  class Agent {
    instance: string;
    targets: number;
    last_update: Date;
  }

  class Rule {
    id: number;
    selector: Array<Array<string>>;
    action: string;
  }

  class Target {
    label_values: Array<string>;
    count: number;
    profiled: number;
    state: string;
  }

  const [targets, setTargets] = React.useState(() => []);

  const [targetsColumns, setTargetsColumns] = React.useState(() => []);

  const [targetsGrouped, setTargetsGrouped] = React.useState<Target[]>(
    () => [],
  );

  const [rules, setRules] = React.useState(() => []);

  const rulesColumnHelper = createColumnHelper<Rule>();
  const rulesColumns = [
    rulesColumnHelper.accessor("id", {
      header: () => "ID",
      cell: (info) => info.getValue(),
    }),
    rulesColumnHelper.accessor("selector", {
      header: () => "Selector",
      cell: (x) =>
        x
          .getValue()
          .map((x) => `${x[0]}${x[1]}"${x[2]}"`)
          .join(", "),
    }),
    rulesColumnHelper.accessor("action", {
      header: () => "Action",
      cell: (info) => info.getValue(),
    }),
    rulesColumnHelper.accessor("id", {
      header: "actions",
      id: "__actions",
      cell: (info) => (
        <Button onClick={() => ruleDelete(info.getValue())} variant="danger">
          <Trash style={{ pointerEvents: "none" }} />
        </Button>
      ),
    }),
  ];

  const [agents, setAgents] = React.useState(() => []);

  const [labelNames, setLabelNames] = React.useState(() => new Set<string>());
  const [labelNamesSelected, setLabelNamesSelected] = React.useState(() => [
    "__container_id__",
  ]);

  const labelNameAdd = (event) => {
    const value = event.target.getAttribute("data");
    setLabelNamesSelected(labelNamesSelected.concat([value]));
  };

  const labelNameRemove = (event) => {
    const value = event.target.getAttribute("data");
    setLabelNamesSelected(labelNamesSelected.filter((x) => x !== value));
  };

  const ruleToggle = (row: Target) => {
    const selector = labelNamesSelected.reduce(
      (o, k, idx) => ((o[k] = row.label_values[idx]), o),
      {},
    );
    if (readyState === ReadyState.OPEN) {
      console.log(
        JSON.stringify({
          type: "rule.toggle",
          selector: JSON.stringify(selector),
        }),
      );
    }
  };

  const ruleDelete = (id: number) => {
    if (readyState === ReadyState.OPEN) {
      console.log(
        JSON.stringify({
          type: "rule.delete",
          payload: { id: id },
        }),
      );
    }
  };

  const agentsColumnHelper = createColumnHelper<Agent>();

  const agentsColumns = [
    agentsColumnHelper.accessor("instance", {
      header: () => "Instance:",
      cell: (info) => info.getValue(),
    }),
    agentsColumnHelper.accessor("targets", {
      header: () => "Targets (count)",
      cell: (info) => info.getValue(),
    }),
    agentsColumnHelper.accessor("last_updated", {
      header: () => "Last Updated",
      cell: (info) => info.getValue(),
    }),
  ];

  // build grouped targets based on selected labels and all targets from agents
  React.useEffect(() => {
    const columnHelper = createColumnHelper<Target>();

    setTargetsColumns(
      labelNamesSelected
        .map((x, idx) =>
          columnHelper.accessor((row) => row.label_values[idx], {
            header: x,
            id: "labels." + x,
          }),
        )
        .concat([
          columnHelper.accessor((row) => row.count, {
            header: "Target Count",
            id: "__count",
          }),
          {
            header: "",
            id: "__actions",
            cell: (e) => (
              <Button
                data-row={e.row.id}
                onClick={() => ruleToggle(e.row.original)}
                variant={e.row.original.button}
              >
                {e.row.original.profiled} Profiled
              </Button>
            ),
          },
        ]),
    );

    // calculate groupBy manually
    const result = Object.groupBy(targets, (x) =>
      labelNamesSelected.map((k) => (x[k] === undefined ? "" : x[k])),
    );
    setTargetsGrouped(
      Object.keys(result).map((x) => {
        const t = new Target();
        t.label_values = labelNamesSelected.map((k) =>
          result[x][0][k] === undefined ? "" : result[x][0][k],
        );
        t.count = result[x].length;
        t.profiled = result[x].length; // TODO: Evaluate
        t.button = "success";
        return t;
      }),
    );
  }, [labelNamesSelected, targets]);

  // handle incoming websocket messages
  React.useEffect(() => {
    if (lastMessage === null) {
      return;
    }
    const msg = JSON.parse(lastMessage.data);
    if ("type" in msg && msg["type"] === "data") {
      if ("payload" in msg && msg["payload"] !== null) {
        const payload = msg["payload"];
        if ("rules" in payload && payload["rules"] !== null) {
          setRules(payload["rules"]);
        }
        if ("agents" in payload && payload["agents"] !== null) {
          setAgents(
            payload["agents"].map((a) => {
              return {
                instance: a.name,
                targets: a.targets.length,
                last_updated: a.last_updated,
              };
            }),
          );
          const labelNames = new Set<string>();
          let targets = [];
          payload["agents"].forEach((a) => {
            targets = targets.concat(a.targets);
            a.targets.forEach((t) => {
              Object.keys(t).forEach((l) => labelNames.add(l));
            });
          });
          setLabelNames(labelNames);
          setTargets(targets);
        }
      }
    }
  }, [lastMessage, setRules, setAgents, setTargets]);

  return (
    <Accordion alwaysOpen defaultActiveKey={["1", "2", "3"]}>
      <Accordion.Item eventKey="0">
        <Accordion.Header>Websocket Status</Accordion.Header>
        <Accordion.Body>
          <h2>Websocket status</h2>
          <p>The WebSocket is currently {readyState}</p>
          {lastMessage ? <p>Last message: {lastMessage.data}</p> : null}
          {rules ? <p>Rules: {JSON.stringify(rules)}</p> : null}
          {targetsGrouped ? (
            <p>Targets Grouped: {JSON.stringify(targetsGrouped)}</p>
          ) : null}
        </Accordion.Body>
      </Accordion.Item>
      <Accordion.Item eventKey="1">
        <Accordion.Header>Rules</Accordion.Header>
        <Accordion.Body>
          <Table columns={rulesColumns} data={rules} />
        </Accordion.Body>
      </Accordion.Item>
      <Accordion.Item eventKey="2">
        <Accordion.Header>Agents</Accordion.Header>
        <Accordion.Body>
          <Table columns={agentsColumns} data={agents} />
        </Accordion.Body>
      </Accordion.Item>
      <Accordion.Item eventKey="3">
        <Accordion.Header>Targets</Accordion.Header>
        <Accordion.Body>
          <Stack direction="horizontal" gap={2}>
            <Dropdown>
              <Dropdown.Toggle variant="success" id="dropdown-basic">
                Select Label
              </Dropdown.Toggle>

              <Dropdown.Menu>
                {[...labelNames]
                  .sort()
                  .filter((x) => !labelNamesSelected.includes(x))
                  .map((x, idx) => (
                    <Dropdown.Item key={idx} data={x} onClick={labelNameAdd}>
                      {x}
                    </Dropdown.Item>
                  ))}
              </Dropdown.Menu>
            </Dropdown>
            {labelNamesSelected.map((x, idx) => (
              <Badge
                pill
                key={idx}
                data={x}
                onClick={labelNameRemove}
                bg="primary"
              >
                {x}
              </Badge>
            ))}
          </Stack>
          <Table columns={targetsColumns} data={targetsGrouped} />
        </Accordion.Body>
      </Accordion.Item>
    </Accordion>
  );
}

const rootElement = document.getElementById("root");
if (!rootElement) throw new Error("Failed to find the root element");

ReactDOM.createRoot(rootElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
