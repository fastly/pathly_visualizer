function EditMeasurements() {

    const startMeasurement = (e) => {
        e.preventDefault()
        var form = new FormData(e.target)

        const formObj = {}
        form.forEach((value, key) => (formObj[key] = value))

        if (formObj.loadHistory === "on") {
            formObj.loadHistory = true
        } else {
            formObj.loadHistory = false
        }

        if (formObj.startLiveCollection === "on") {
            formObj.startLiveCollection = true
        } else {
            formObj.startLiveCollection = false
        }

        sendPostRequest(formObj, true)
    }

    const stopMeasurement = (e) => {
        e.preventDefault()
        var form = new FormData(e.target)

        const formObj = {}
        form.forEach((value, key) => (formObj[key] = value))

        if (formObj.dropStoredData === "on") {
            formObj.dropStoredData = true
        } else {
            formObj.dropStoredData = false
        }

        sendPostRequest(formObj, false)

    }

    const sendPostRequest = (requestObj, start) => {
        requestObj.atlasMeasurementId = parseInt(requestObj.atlasMeasurementId)
        let fetchUrl = "http://localhost:8080/api/measurement/start"
        if (!start) {
            fetchUrl = "http://localhost:8080/api/measurement/stop"
        }
        console.log(requestObj)

        fetch(fetchUrl, {
            method: 'POST',
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify(requestObj),
        })
        .then((response) => {
            if(response.status === 200){
                alert("Success!")
            } else if (response.status === 400 && start){
                alert("An error occurred. Please ensure the Measurement ID is correct and that one of 'Fetch Historical Data' or 'Start Live Collection' is selected.")
            } else if (response.status === 500 && !start){
                alert("An error occurred. Measurement can't be stopped as it's not being tracked live, if 'Drop Stored Data' is checked, historical data of that measurement will be dropped")
            } else {
                alert("An error occurred. Please ensure all information is correct.")
            }
            window.location.reload()
        })
        .catch((error) => {
            console.error(error)
        })
            .then((response) => response.json())
            .then((data) => {
                console.log(data)
            })
            .catch((error) => {
                console.error(error)
            })
    }

    const getMeasurements = () => {
        let fetchUrl = "http://localhost:8080/api/measurement/list"
        fetch(fetchUrl)
        .then((response) => response.json())
        .then((data) => {
            document.getElementById("listOfMeasurements").innerHTML = JSON.stringify(data)
        })
        .catch((error) => {
            console.error(error)
        })
    }

    return (
        <>
        <div className="editMeasurements">
            <div className="formDiv">
                <form onSubmit={startMeasurement}>
                    <h1>START TRACKING MEASUREMENT</h1>
                    <label for="atlasMeasurementId">RIPE Atlas Measurement ID</label>
                    <input name="atlasMeasurementId" placeholder="e.g. 123456" style={{width: "90%"}} required></input>
                    <br/><br/>
                    <label for="loadHistory">Fetch Historical Data</label>
                    <input name="loadHistory" type={"checkbox"} style={{margin: 10}}></input>
                    <br/>
                    <label for="startLiveCollection">Start Live Collection</label>
                    <input name="startLiveCollection" type={"checkbox"} style={{margin: 10}}></input>
                    <br/>
                    <button className="submitForm" type="submit">Submit</button>
                </form>
            </div>
            <div className="formDiv">
                <form onSubmit={stopMeasurement}>
                    <h1>STOP TRACKING MEASUREMENT</h1>
                    <label for="atlasMeasurementId">RIPE Atlas Measurement ID</label>
                    <input name="atlasMeasurementId" placeholder="e.g. 123456" style={{width: "90%"}} required></input>
                    <br/><br/>
                    <label for="dropStoredData">Drop Stored Data</label>
                    <input name="dropStoredData" type={"checkbox"} style={{margin: 10}}></input>
                    <br/>
                    <button className="submitForm" type="submit">Submit</button>
                </form>
            </div>
        </div>
        <div style={{position: "absolute", top: "60%", width: "100%", textAlign: "center"}}>
            <button className="submitForm" type="button" onClick={getMeasurements}>Check Measurement List</button>
            <p id="listOfMeasurements">List of Measurements Here</p>
        </div>
        </>
    )
}

export default EditMeasurements