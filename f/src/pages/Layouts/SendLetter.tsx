import ButtonInput from "@/components/ButtonInput"
import InputField from "@/components/InputField"
import { useState } from "react"
import styles from "@/styles/HomePage.module.css"
import BroadcastRequest from "@/types/BroadcastRequest"
import Broadcast from "../api/Email-APIs/broadcast"
import SetAnnouncement from "../api/Email-APIs/SetAnnouncement"
import Announcement from "@/types/Announcement"

const SendLetter = () =>
{

  // message broadcast 
  const [message, setMessage] = useState("")

  const handleFormSubmit = async (e: any) =>
  {
    e.preventDefault()
    console.log("You are trying to broadcast " + message + " to all mail subscriber")

    const newBroadcast:BroadcastRequest = {
      Message: message
    }

    const response = await Broadcast(newBroadcast)
    console.log(response)

    const setAnnouncementReq:Announcement = {
      announcement_message : message
    }

    const saveResponse = await SetAnnouncement(setAnnouncementReq)

    if (response !== -1 && response.data.message === "Email sent successfully") {
      alert("Email successfully broadcasted!")
      window.location.reload()
    }

  }

  return (
    <div className={styles.SendLetter}>
      <h1>
        Broadcast A Newsletter
      </h1>
      <form className={styles.emailform} onSubmit={handleFormSubmit}>
        <InputField width={800} onChange={setMessage} value={message}  placeholder="Message here..." />
        <ButtonInput blue placeholder="Broadcast now" submit />
      </form>
    </div>
  )
}

export default SendLetter;