import React from "react";
import ReactDOM from "react-dom/client";
import useWebSocket, { ReadyState } from "react-use-websocket";
const WS_URL = "ws://127.0.0.1:8333/ws";

import Table from "./table";
import "./index.css";

import { useReactTable, createColumnHelper } from "@tanstack/react-table";
import Accordion from "react-bootstrap/Accordion";
import Dropdown from "react-bootstrap/Dropdown";
import Badge from "react-bootstrap/Badge";
import Stack from "react-bootstrap/Stack";

import { makeData, Person } from "./makeData";

import "bootstrap/dist/css/bootstrap.min.css";

function App() {
  const [socketUrl, setSocketUrl] = React.useState(WS_URL);
  const didUnmount = React.useRef(false);

  const { sendMessage, lastMessage, readyState } = useWebSocket(socketUrl, {
    shouldReconnect: (closeEvent) => {
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

  const rerender = React.useReducer(() => ({}), {})[1];

  const [rules, setRules] = React.useState(() => []);

  type Rule = {
    ID: number;
    //    Selector: string;
    Action: string;
  };

  const rulesColumnHelper = createColumnHelper<Rule>();

  const rulesColumns = [
    rulesColumnHelper.accessor("ID", {
      header: () => "ID",
      cell: (info) => info.getValue(),
      footer: (info) => info.column.id,
    }),
    rulesColumnHelper.accessor("Selector", {
      header: () => "Selector",
      cell: (info) => info.getValue(),
      footer: (info) => info.column.id,
    }),
    rulesColumnHelper.accessor("Action", {
      header: () => "Action",
      cell: (info) => info.getValue(),
      footer: (info) => info.column.id,
    }),
  ];

  const rulesTable = useReactTable({
    rules,
    rulesColumns,
  });

  type Agent = {
    Instance: string;
    Targets: number;
    LastUpdate: Date;
  };

  type Targets = {
    Instance: string;
    Filtered: bool;
  };

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

  const agentsTable = useReactTable({
    agents,
    agentsColumns,
  });

  const [targets, setTargets] = React.useState(() => []);

  const [targetsColumns, setTargetsColumns] = React.useState(() => []);

  const [targetsGrouping, setTargetsGrouping] = React.useState<GroupingState>(
    [],
  );

  React.useEffect(() => {
    const columnHelper = createColumnHelper<{ [key: string]: string }>();

    setTargetsColumns(
      labelNamesSelected
        .map((x) =>
          columnHelper.accessor(x, {
            header: x,
            id: "label." + x,
          }),
        )
        .concat([
          {
            accessorKey: "count",
            header: () => "Count",
            aggregationFn: "count",
            //                aggregatedCell: ({ getValue }) => getValue().toLocaleString(),
          },
        ]),
    );

    /*
      calculate manually
    let result = {};
    targets.forEach((t) =>{
      let o = result;
      labelNamesSelected.forEach((f) => {
       if(o[f] === undefined)
         o[f] = {};
        o = o[t[f]]
      });
      o++;
    });
      */

    setTargetsGrouping(labelNamesSelected.map((x) => "label." + x));
  }, [labelNamesSelected, targets, setTargetsGrouping, setTargetsColumns]);

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
          let labelNames = new Set<string>();
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

  const [data, setData] = React.useState(() => makeData(100000));
  const refreshData = () => setData(() => makeData(100000));

  return (
    <Accordion alwaysOpen defaultActiveKey={["1", "2", "3"]}>
      <Accordion.Item eventKey="0">
        <Accordion.Header>Websocket Status</Accordion.Header>
        <Accordion.Body>
          <h2>Websocket status</h2>
          <p>The WebSocket is currently {readyState}</p>
          {lastMessage ? <p>Last message: {lastMessage.data}</p> : null}
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
                  .map((x) => (
                    <Dropdown.Item data={x} onClick={labelNameAdd}>
                      {x}
                    </Dropdown.Item>
                  ))}
              </Dropdown.Menu>
            </Dropdown>
            {labelNamesSelected.map((x) => (
              <Badge pill data={x} onClick={labelNameRemove} bg="primary">
                {x}
              </Badge>
            ))}
          </Stack>
          <Table
            columns={targetsColumns}
            data={targets}
            grouping={targetsGrouping}
            setGrouing={setTargetsGrouping}
          />
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
