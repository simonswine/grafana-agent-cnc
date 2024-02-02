import React from "react";
import {
  getCoreRowModel,
  getExpandedRowModel,
  getGroupedRowModel,
  getPaginationRowModel,
  getFilteredRowModel,
  useReactTable,
  flexRender,
} from "@tanstack/react-table";

import { Table as BTable } from "react-bootstrap";

export default function Table({
  columns,
  data,
  grouping = null,
  setGrouping = null,
}) {
  const rerender = React.useReducer(() => ({}), {})[1];

  const opts =
    grouping == null && setGrouping == null
      ? {}
      : {
          state: {
            grouping,
          },
          onGroupingChange: setGrouping,
          getExpandedRowModel: getExpandedRowModel(),
          getGroupedRowModel: getGroupedRowModel(),
          //          getPaginationRowModel: getPaginationRowModel(),
          getFilteredRowModel: getFilteredRowModel(),
        };

  // Use the useTable Hook to send the columns and data to build the table
  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
    ...opts,
  });

  return (
    <div className="p-2">
      <BTable striped bordered hover responsive size="sm">
        <thead>
          {table.getHeaderGroups().map((headerGroup) => (
            <tr key={headerGroup.id}>
              {headerGroup.headers.map((header) => (
                <th key={header.id}>
                  {header.isPlaceholder
                    ? null
                    : flexRender(
                        header.column.columnDef.header,
                        header.getContext(),
                      )}
                </th>
              ))}
            </tr>
          ))}
        </thead>
        <tbody>
          {table.getRowModel().rows.map((row) => (
            <tr key={row.id}>
              {row.getVisibleCells().map((cell) => (
                <td key={cell.id}>
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </BTable>
      <div>
        <button onClick={() => rerender()}>Force Rerender</button>
      </div>
    </div>
  );
}
