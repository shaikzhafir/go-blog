import { useState } from 'react'
import reactLogo from './assets/react.svg'
import viteLogo from '/vite.svg'
import './App.css'

function App() {
  const [count, setCount] = useState(0)

  return (
    <>
        <div class="py-20 mx-5 xl:mx-80 bg-slate-500 mb-10 font-geist">
    <p class="text-3xl my-2 text-green-500">welcome to szhafir blog</p>
    <h1>Reading Now</h1>
    <div class="grid grid-cols-1 md:grid-cols-4 gap-8">
    </div>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-8">
      <div>
        <h1>Book Reviews</h1>
        <div id="book-reviews" class="h-full">
          <div class="htmx-indicator">
            loading... notion api is slow.. give chance..
          </div>
        </div>
      </div>
      <div>
        <div>
          <h1>Coding Posts</h1>
          <div id="coding-posts" class="h-full">
            <div class="htmx-indicator">
              loading... notion api is slow.. give chance..
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
    </>
  )
}

export default App
