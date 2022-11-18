import { useState } from 'react';
import './App.css';
import Graph from './components/Graph';

//test data for graph population
import {tData} from './components/testTrcrt'
import {tDataFull} from './components/testTrcrtFull'

function App() {

  // list of graphs being rendered --> start with state []
  const [graphList, setGraphList] = useState([])

  // Search function --> connect to rest API
  // Once RIS data collection is implemented, this will include that as well
  const search = (e) => {
    e.preventDefault()

    // get serialized form data
    var form = new FormData(e.target)

    // convert form data into object to send in http rq
    const formObj = {}
    form.forEach((value, key) => (formObj[key] = value))

    // split into two form obj to make two http requests
    let ipSplit = formObj.destinationIp.split(" / ")
    const ipv4FormObj = 
      {
        probeId: formObj.probeId,
        destinationIp: ipSplit[0],
      }
    const ipv6FormObj = 
      {
        probeId: formObj.probeId,
        destinationIp: ipSplit[1],
      }

    // create new http rq --> note, I've used XMLHttpRequest before but if there's a preferred method of sending requests use that instead
    const xhr = new XMLHttpRequest()
    // I assume keeping localhost here is fine as the code will be running on GCP regardless
    if(document.getElementById("fullOrClean").checked){
      xhr.open("POST", "http://localhost:8080/api/traceroute/clean", true)
    }
    else{
      xhr.open("POST", "http://localhost:8080/api/traceroute/full", true)
    }
    xhr.setRequestHeader("Content-Type", "application/json")
    // what happens when response is received
    xhr.onreadystatechange = () => {
      if(xhr.readyState === XMLHttpRequest.DONE && xhr.status === 200) {
        console.log(xhr.response)
        //concat graph onto current graph list, gets rerendered w/ new graph list
        setGraphList(graphList.concat(<Graph response={xhr.response} form={formObj} clean={document.getElementById("fullOrClean").checked}></Graph>))
      }
    }

    //two requests for ipv4 and 6 addresses
    xhr.send(JSON.stringify(ipv4FormObj))
    xhr.send(JSON.stringify(ipv6FormObj))

    // TODO --> once receiving responses, concat all responses together into one large response to be able to represent ipv4 and 6 on same graph
  }

  return (
    <>
      <div className="App">
        {/* FORM FOR COLLECTING DATA FROM BACKEND STORAGE HERE */}
        <h1>CREATE VISUALIZATION</h1>
        <form id="postForm" onSubmit={search}>
        <label for="src">Source Probe</label>
          <input id= "srcProbe" name="probeId" placeholder="e.g. 123456" required></input>
          <br></br>
          {/* Using list of measurements sds suggested to start from */}
          <label for="dst">Destination IP</label>
          <select id="destIP" name="destinationIp" placeholder="Destination IP" required>
          <option hidden> Select IP Address</option>
            <optgroup label="k-root">
              <option value="193.0.14.129 / 2001:7fd::1">193.0.14.129 / 2001:7fd::1</option>
            </optgroup>
            <optgroup label="b-root">
              <option value="199.9.14.201 / 2001:500:200::b">199.9.14.201 / 2001:500:200::b</option>
            </optgroup>
            <optgroup label="fastly anycast">
              <option value="151.101.0.1 / 2a04:4e42::1">151.101.0.1 / 2a04:4e42::1</option>
            </optgroup>
          </select>
          <br></br>
          <div className="switchBox">
            <p>Full Data</p>
            <label className="switch">
              <input type="checkbox" name="fullOrClean" id="fullOrClean"/>
              <span className="slider round"></span>
            </label>
            <p>Clean Data</p>
          </div>
          <br></br>
          <button id="submitForm" type="submit">Visualize</button>
        </form>
      </div>
      {/* Left empty, graphs rendered on response load */}
      <div id="graphArea">
        {/* {graphList} */}
        {/* <Graph response={tDataFull} clean={false}></Graph> */}
        <Graph response={tData} clean={true}></Graph>
      </div>
    </>
  );
}

export default App;
