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
          // Compare the new data with the previous block
          if (JSON.stringify(data) !== JSON.stringify(blocks[0])) {
            // Data is different, so append the new block to the existing list of blocks
            setBlocks(prevBlocks => [data, ...prevBlocks]);
          }
        })
        .catch((error) => {
          console.error("Error fetching latest block:", error);
        });
    };
  
    // Fetch data initially
    // fetchData();
    // console.log("Dataaa: ",blocks);
  
    // Fetch data every 5 seconds
    const intervalId = setInterval(fetchData, 5000);
  
    // Cleanup function to clear the interval when the component unmounts
    return () => clearInterval(intervalId);
  }, [blocks]); // Include blocks in the dependency array to trigger the effect when blocks change
  

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
