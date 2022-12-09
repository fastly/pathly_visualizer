// import { Text } from '@react-ui-org/react-ui';

function About() {
    return (

        <>
            {/* <p classname="logos">Test</p><p><img src={require("./images/fastly_logo.png")} width={500}></img></p> */}
            <div className="aboutPage">
                <h2> This project is sponsored by: </h2>
                <div className="sponsors">                
               
                <p classname="logos"></p><p><img src={require("./images/fastly_logo.png")} width={400}></img></p>
                <p classname="logos"></p><p><img src={require("./images/wpi_logo.png")} width={400}></img></p>
                </div>
            </div>
        </>
  
    )
}

export default About