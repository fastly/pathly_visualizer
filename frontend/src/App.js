import { useState } from 'react';
import './App.css';
import Graph from './components/Graph';

//test data for graph population
import {tData} from './components/test_data/testTrcrt'
import {tDataFull} from './components/test_data/testTrcrtFull'

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

    const sendingObjs = ipFormObjs(ipSplit, formObj)
    
    fetchData(sendingObjs, formObj)
    
  }

  const fetchData = (objs, formObj) => {

    let combProbeIp = "";
    let combNodes = [];
    let combEdges = [];
    let combinedResponse

    let done = false

    let clean = document.getElementById("fullOrClean").checked

    let fetchUrl = "http://localhost:8080/api/traceroute/full"
    if(clean){
      fetchUrl = "http://localhost:8080/api/traceroute/clean"
    }

    const requests = objs.map(ipObj => {
      console.log(ipObj)
      fetch(fetchUrl, {
        method: 'POST',
        headers: {
          "Content-Type": "application/json"
        },
        body: JSON.stringify(ipObj),
      })
      .then((response) => response.json())
      .then((data) => {
        console.log(data)
        combNodes.push(...data.nodes)
        combEdges.push(...data.edges)
        if(combProbeIp === ""){
          // probeIps in clean vs. probeIds in full
          if(clean){
            for(const probe of data.probeIps){
              combProbeIp += probe
            }
          }
          else{
            for(const probe of data.probeIds){
              combProbeIp += probe.ip
            }
          }
        } else {
          // probeIps in clean vs. probeIds in full
          if(clean){
            for(const probe of data.probeIps){
              combProbeIp += " / " + probe
            }
          }
          else{
            for(const probe of data.probeIds){
              combProbeIp += " / " + probe.ip
            }
          }
          done = true
        }
      })
      .catch((error) => {
        console.error(error)
      })
      .then(function() {
        combinedResponse = {
          probeIp: combProbeIp,
          nodes: combNodes,
          edges: combEdges,
        }
        console.log(combinedResponse)
        if(done){
          setGraphList(graphList.concat(
            <Graph response={combinedResponse} form={formObj} clean={document.getElementById("fullOrClean").checked} asnSetting={document.getElementById("boxOrColor").checked}>
            </Graph>
          ))
        }
      })
    })

    Promise.all(requests).then(console.log("test"))
    
  }

  const ipFormObjs = (splitArr, formObj) => {
    let formObjs = []
    for(let i = 0; i < splitArr.length; i++){
      const obj = {
        probeId: parseInt(formObj.probeId),
        destinationIp: splitArr[i],
      }
      formObjs.push(obj)
    }

    return formObjs
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
          <div className="switchBox">
            <p>ASN Boxes</p>
            <label className="switch">
              <input type="checkbox" name="boxOrColor" id="boxOrColor"/>
              <span className="slider round"></span>
            </label>
            <p>ASN Colors</p>
          </div>
          <button id="submitForm" type="submit">Visualize</button>
        </form>
      </div>
      <div id="graphArea">
        {graphList}
        {/* <Graph response={tDataFull} clean={false}></Graph> */}
        {/* <Graph response={tData} clean={true}></Graph> */}
      </div>
    </>
  );
}

export default App;
