import React, { useEffect, useState } from "react";
import "./App.css";
<link
  href="https://fonts.googleapis.com/css? family=Open+Sans:300, 400, 700"
  rel="stylesheet"
  type="text/css"
/>;

function YourComponent() {
  const [blocks, setBlocks] = useState([]);

  useEffect(() => {
    const fetchData = () => {
      fetch("http://localhost:3001/latest-block-number")
        .then((res) => res.json())
        .then((data) => {
          // Append the new block to the existing list of blocks
          console.log(data);
          setBlocks((prevBlocks) => [...prevBlocks, data]);
        });
    };

    // Fetch data initially
    fetchData();

    // Fetch data every 10 seconds
    const intervalId = setInterval(fetchData, 36000);

    // Cleanup function to clear the interval when the component unmounts
    return () => clearInterval(intervalId);
  }, []); // Empty dependency array ensures the effect runs only once after initial render

  return (
    <div className="main_container">
      <h1 className="title">Latest Blocks</h1>
      <table className="container">
        <thead>
          <tr>
            <th>
              <h1>Serial Number</h1>
            </th>
            <th>
              <h1>Block Number</h1>
            </th>
            <th>
              <h1>Block Hash</h1>
            </th>
            <th>
              <h1>Time</h1>
            </th>
          </tr>
        </thead>
        <tbody>
          {blocks.map((block, index) => (
            <tr key={index}>
              <td>{index + 1}</td>
              <td>{block.latestBlockNumber}</td>
              <td>{block.latestBlockHash}</td>
              <td>{block.timestamp}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export default YourComponent;
