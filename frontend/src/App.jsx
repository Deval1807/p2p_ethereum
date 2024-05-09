import { useEffect, useState } from "react";

import "./App.css";

function App() {
  const [column, setColumn] = useState([]);
  const [records, setRecords] = useState([]);

  useEffect(() => {
    fetch("")
      .then((res) => res.json())
      .then((data) => {
        setColumn(Object.keys(data.users[0]));
        setRecords(data.users);
      });
  }, []);

  return (
    <>
      <div>
        <table>
          <thread>
            <tr>
              {column.map((c, i) => (
                <th key={i}>{c}</th>
              ))}
            </tr>
          </thread>
          <tbody>
            {records.map((record, i) => (
              <tr key={i}>
                <td>{record.latestblock}</td>
                <td>{record.latestblockhash}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
}

export default App;
