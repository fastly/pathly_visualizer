
function About() {
    return (

        <>
            <div className="aboutPage">

                <p className="logos"></p><p><img src={require("./images/pathly_logo_white.png")} width={500}></img></p>
                <h4 className="descrip"> PATHly (Probe ASN Traceroute Highway) Visualizer is a web-based traceroute pathing tool to visualize paths that packets take from a source probe to Fastly Anycast destinations. PATHly Visualizer allows users to interactively display traceroute data and better understand and evaluate Anycast forwarding in IPv4 and IPv6.
                </h4>

                <br></br>
                <h4 >PATHly Visualizer was developed by a team of students at <br></br><a href="https://www.wpi.edu/">Worcester Polytechnic Institute</a> for their <a href="https://www.wpi.edu/academics/undergraduate/major-qualifying-project">Major Qualifying Project:</a></h4>
                <br></br>
                <h3>Jack Campanale '23
                    <br></br>
                    Jasper Meggitt '23
                    <br></br>
                    Yash Patel '23
                    <br></br>
                    Cindy Trac '23
                </h3>
                <br></br>

                <p className="logos"></p><p><img src={require("./images/spacer.png")} width={80}></img></p>
                <br></br>
                <h3> This project was sponsored by:  </h3>
                <div className="sponsors">

                    <br></br>
                    <p className="logos"></p><p><img src={require("./images/fastly_logo.png")} width={300}></img></p>
                    <p className="logos"></p><p><img src={require("./images/wpi_logo.png")} width={300}></img></p>
                </div>
                <br></br>

            </div>
        </>

    )
}

export default About