import React from "react";
import ReactDOM from "react-dom/client";
import useWebSocket, { ReadyState } from "react-use-websocket";
const WS_URL = "ws://127.0.0.1:8333/ws";

import Table from "./table";
import "./index.css";

import { useReactTable, createColumnHelper } from "@tanstack/react-table";
import { makeData, Person } from "./makeData";

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
    Selector: string;
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

  type Targets = {
    Instance: string;
    Filtered: bool;
  };

  const [targets, setTargets] = React.useState(() => []);

  const targetsColumnHelper = createColumnHelper<Target>();

  const targetsColumns = [
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

  const targetsTable = useReactTable({
    targets,
    targetsColumns,
  });

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
        if ("targets" in payload && payload["targets"] !== null) {
          setTargets(payload["targets"]);
        }
      }
    }
  }, [lastMessage, setRules, setTargets]);

  const [data, setData] = React.useState(() => makeData(100000));
  const refreshData = () => setData(() => makeData(100000));

  const [grouping, setGrouping] = React.useState<GroupingState>([]);

  return (
    <div>
      <h1>Grafana Agent Command and Control</h1>
      <div id="status">
        <h2>Websocket status</h2>
        <p>The WebSocket is currently {readyState}</p>
        {lastMessage ? <p>Last message: {lastMessage.data}</p> : null}
      </div>
      <div id="rules">
        <h2>Rules</h2>
        <Table columns={rulesColumns} data={rules} />
      </div>
      <div id="targets">
        <h2>Targets</h2>
        <Table columns={targetsColumns} data={targets} />
      </div>
    </div>
  );
}

const rootElement = document.getElementById("root");
if (!rootElement) throw new Error("Failed to find the root element");

ReactDOM.createRoot(rootElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
