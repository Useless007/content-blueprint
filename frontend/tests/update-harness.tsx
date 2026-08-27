import React from 'react'
import {createRoot} from 'react-dom/client'
import '../src/style.css'
import '../src/onboarding/onboarding.css'
import {UpdateCenter} from '../src/update/UpdateCenter'
import './update-harness.css'

createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <main className="update-harness">
      <span>Content Blueprint</span>
      <h1>Update Center browser smoke test</h1>
      <p>พื้นหลังจำลองเพื่อทดสอบ modal, banner, keyboard focus และ responsive layout</p>
    </main>
    <button className="onboarding-help-button" type="button">วิธีใช้</button>
    <UpdateCenter />
  </React.StrictMode>,
)
