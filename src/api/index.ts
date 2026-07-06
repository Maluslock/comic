import { mockPhotographers, mockServices, mockReviews, mockWorks, mockEvents, tags, timeSlots } from '@/data/mock'
import type { Photographer, Service, Review, Work, ComicEvent } from '@/types'

const BASE_URL = 'https://api.example.com'

function delay(ms: number) {
  return new Promise(resolve => setTimeout(resolve, ms))
}

export async function getPhotographers(params?: {
  keyword?: string
  tags?: string[]
  location?: string
  page?: number
  size?: number
}): Promise<{ list: Photographer[]; total: number }> {
  await delay(500)
  let list = [...mockPhotographers]
  
  if (params?.keyword) {
    const kw = params.keyword.toLowerCase()
    list = list.filter(p => 
      p.name.toLowerCase().includes(kw) || 
      p.description?.toLowerCase().includes(kw)
    )
  }
  
  if (params?.tags?.length) {
    list = list.filter(p => 
      p.tags.some(t => params!.tags!.includes(t))
    )
  }
  
  if (params?.location) {
    list = list.filter(p => p.location === params.location)
  }
  
  return {
    list: list.slice((params?.page || 1) - 1, (params?.page || 1) * (params?.size || 10)),
    total: list.length
  }
}

export async function getPhotographerById(id: string): Promise<Photographer> {
  await delay(300)
  const photographer = mockPhotographers.find(p => p.id === id)
  if (!photographer) {
    throw new Error('摄影师不存在')
  }
  
  return {
    ...photographer,
    services: mockServices,
    reviews: mockReviews.filter(r => r.photographerId === id),
    works: mockWorks.filter(w => w.photographerId === id)
  }
}

export async function getWorks(photographerId?: string): Promise<Work[]> {
  await delay(300)
  if (photographerId) {
    return mockWorks.filter(w => w.photographerId === photographerId)
  }
  return mockWorks
}

export async function getServices(photographerId?: string): Promise<Service[]> {
  await delay(300)
  return mockServices
}

export async function getReviews(photographerId: string): Promise<Review[]> {
  await delay(300)
  return mockReviews.filter(r => r.photographerId === photographerId)
}

export async function getTags(): Promise<string[]> {
  await delay(200)
  return tags
}

export async function getTimeSlots(date: string): Promise<string[]> {
  await delay(200)
  return timeSlots
}

export async function createBooking(data: {
  photographerId: string
  coserId: string
  serviceId: string
  date: string
  time: string
  remarks?: string
}): Promise<{ id: string }> {
  await delay(500)
  return { id: `booking_${Date.now()}` }
}

export async function getBookings(userId: string, status?: string): Promise<any[]> {
  await delay(500)
  return []
}

export async function getComicEvents(params?: {
  location?: string
  status?: string
}): Promise<ComicEvent[]> {
  await delay(300)
  let list = [...mockEvents]
  if (params?.location) {
    list = list.filter(e => e.location === params.location)
  }
  if (params?.status) {
    list = list.filter(e => e.status === params.status)
  }
  return list.sort((a, b) => a.startDate - b.startDate)
}
