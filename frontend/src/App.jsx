import React, { useEffect, useState } from 'react';

function YourComponent() {
  const [blocks, setBlocks] = useState([]);

  useEffect(() => {
    const fetchData = () => {
      fetch("http://localhost:3001/latest-block-number")
        .then((res) => res.json())
        .then((data) => {
          // Append the new block to the existing list of blocks
          console.log(data);
          setBlocks(prevBlocks => [...prevBlocks, data]);
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
    <div>
      <h1>Latest Blocks</h1>
      <table>
        <thead>
          <tr>
            <th>Block Number</th>
            <th>Block Hash</th>
          </tr>
        </thead>
        <tbody>
          {blocks.map((block, index) => (
            <tr key={index}>
              <td>{block.latestBlockNumber}</td>
              <td>{block.latestBlockHash}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export default YourComponent;
