use std::time::Duration;
use format_serde_error::SerdeError;
use reqwest::Client;
use serde_json::Value;
use tokio::io::AsyncWriteExt;
use crate::{MeasurementResponse, UsageLimiter};

async fn start() {
    let client = UsageLimiter::new(Client::new(), 10, Duration::from_secs(1));
    download_measurements(&client, "https://atlas.ripe.net:443/api/v2/measurements/traceroute/").await.unwrap();

    // const URL: &'static str = "https://atlas.ripe.net:443/api/v2/measurements/traceroute/";
    // // const URL: &'static str = "https://atlas.ripe.net/api/v6/measurements/";
    //
    // // let response = reqwest::get(format!("{}?page_size=500&type=traceroute", URL))
    // let response = reqwest::get("https://atlas.ripe.net/api/v2/measurements/5001/results/")
    //     .await
    //     .unwrap()
    //     .text()
    //     .await
    //     .unwrap();
    //
    //
    //
    // println!("Got response!");
    // let mut file = BufWriter::new(File::create("response.json").expect("Unable to create file"));
    // let value = match serde_json::from_str::<Value>(&response) {
    //     Ok(v) => v,
    //     Err(e) => {
    //         println!("{}", &response);
    //         println!("Response is not valid json!");
    //         panic!("{:?}", e);
    //     }
    // };
    // serde_json::to_writer_pretty(&mut file, &value).expect("wrote data to file");
    //
    //
    // // let parsed = serde_json::from_str::<MeasurementResponse>(&response).unwrap();
    // let parsed = from_json_str::<MeasurementResponse>(&response);
    // // let json = serde_json::from_str::<Value>(&response).unwrap();
    //
    // // dbg!(parsed);
    // match parsed {
    //     Ok(v) => {
    //         println!("Received {} results", v.results.len());
    //
    //     },
    //     Err(e) => {
    //         // let value = serde_json::from_str::<Value>(&response);
    //
    //         println!("{}", e)
    //     },
    // }

    // println!("{:?}", parsed);
    // println!("{}", serde_json::to_string_pretty(&json).unwrap());
}

// fn build_client() -> RateLimit<Client> {
//     let client = Client::new();
//
//     ServiceBuilder::new()
//         .rate_limit(10, Duration::from_secs(1))
//         .service(client)
// }

async fn download_measurements<S: AsRef<str>>(client: &UsageLimiter<Client>, url: S) -> anyhow::Result<()> {
    // let response = reqwest::get(url.as_ref()).await?.text().await?;
    let response = client.perform_rate_limited(|client| client.get(url.as_ref()).send())
        .await?.text().await?;

    let parsed = serde_json::from_str::<MeasurementResponse>(&response).map_err(|err| SerdeError::new(response.to_owned(), err))?;

    let buffer = serde_json::to_vec_pretty(&parsed)?;
    let mut file = tokio::io::BufWriter::new(tokio::fs::File::create("raw_data/measurements.json").await?);
    file.write_all(&buffer).await?;

    let mut buffered_requests = Vec::with_capacity(500);

    for result in &parsed.results {
        // if let Some(id) = result.id {
        buffered_requests.push(fetch_single(client, format!("{}", result.id)));
        // }
    }

    println!("Sent {} requests", buffered_requests.len());

    for request in buffered_requests {
        if let Err(e) = request.await {
            eprintln!("Found error on request: {:?}", e);
        }
    }

    Ok(())
}


async fn fetch_single(client: &UsageLimiter<Client>, pk: String) -> anyhow::Result<()> {
    let url = format!("https://atlas.ripe.net/api/v2/measurements/{}/results/", pk);
    // let response_text = reqwest::get(url).await?.text().await?;
    // let x = client.ready_and().await;
    let response_text = client.perform_rate_limited(|client| client.get(url).send())
        .await?.text().await?;

    match serde_json::from_str::<Value>(&response_text) {
        Ok(v) => {
            let path = format!("raw_data/results_{}.json", pk);
            let buffer = serde_json::to_vec_pretty(&v)?;

            let mut file = tokio::io::BufWriter::new(tokio::fs::File::create(&path).await?);
            file.write_all(&buffer).await?;
            println!("pk {}: Saved response", pk);

        },
        Err(e) => {
            println!("pk {}: Response not json: {:?}", pk, e);
        }
    }

    Ok(())
}
