# Audio Lifelog Processing System

I got a [limitless.ai](http://limitless.ai) pendant, with one year unlimited transcription. I’d like to experiment with making my own system using their data.  Help me make a plan for this, do not start implementing anything, just planning.  These are my thoughts, i want your review of their soundness, completeness, etc, and i want your feedback on best approach/practice \- especially where you see room for improvement \- i want the best solution, these are just my initial thoughts on them.

# Development

* I have used cursor ide for months, but want to try google antigravity for this project  
* I want to try using https://github.com/steveyegge/beads for this project  
* I prefer golang where it is a good fit (eg back-end)  
* For front-end, I’d like cross-platform (web, android, ios), maybe flutter (but welcome recommendations).  I’ve been around html/css/js/ts but not written much myself.  
* I am very experienced with sql, but have light experience with no-sql (can use it if it is the best architectural choice)  
* See my resume [here](https://drive.google.com/file/d/15YleUGyFx_mTcKLMPzV01dRIsZWjoS2g/view?usp=drive_link) for background info

# Hardware

* I have a 2021 M1 macbook pro with 16G.  
* I am familiar with AWS, so would like to use that initially, but may like to move to something else later (eg google) so there should be a clean separation between functionality and hosting/deployment.  
* I am building this as a personal experiment/service for now, but I’d like to architect it to possibly become a service for others later, or maybe as a way to get audio\&transcripts into other systems such as [https://fulcradynamics.github.io/developer-docs/](https://fulcradynamics.github.io/developer-docs/) or [https://gyrosco.pe/one/](https://gyrosco.pe/one/) (if they have an API?)

# Get Data

I have a [limitless.ai](http://limitless.ai) pendant and 1-year unlimited subscription, so can use their api [https://www.limitless.ai/developers](https://www.limitless.ai/developers) to get data (subject to rate limiting), and see openai.yml which i downloaded.  My api key is in .env (that could be renamed if there is a better practice/convention), other vars should be added there eg the api host (perhaps I’ll want to make an api that matches theirs, and then i can extend it for experimentation).

## Audio

* get all audio logs in 1hr chunks (for simplicity), probably saving to files like YYYY/MM/DD/HH.ogg.  
* implement in go, so i can run it locally on macbook (saving to disk), or in AWS lambda later (possibly triggered by eventbridge) saving to s3.  It could be a command-line utility with arguments to grab a date range (eg the tool could consult the database).  
* I’m thinking a fast simple database is appropriate locally (eg sqlite), and maybe RDS later (if it becomes a service rather than a personal experiment).  
* A sample audio is in audio.ogg

## Transcripts

Probably also get the transcripts, although i’m sure of the storage format/granularity/etc.  Needs work…  these come in the form of lifelogs, see lifelog.json as an example

# Associate Contacts

Limitless does provides a way to associate names with voices, but i don’t love the current offering, which is basically to use the app to review the log, select a portion of the transcript, play it, assign it to a name, which I think is only applied within the conversation it is a part of, and then future recordings (not historically).  

## Contacts

I instead want something like

* web/android/ios page that has three areas as below.  The purpose is to be able to associate contacts with their voice, including over the last days/weeks and have it apply to all recordings of that person (not just future ones).  The current method requires that you identify voices in the app very soon after they are recorded (so you get future labelled), and the UI isn’t awesome, and i saw a request from a user to be able to do this via the web page (so this would fill that need, eg its not just me asking).  
  * Contacts \- pulls from (integrated with) your contacts (google, apple), shows a simple table eg with columns {picture, name, known}, where the last is a green check if the speaker voice is known, and a red X otherwise.  Ability to filter by known/unknown/both.  Clicking on a known speaker will display their recordings in the Recordings area.  It’d be nice to be able to search for a contact by name too, which would show the set of matching names.  
  * Unknown speakers \- simple table with columns {id, latest, recordings, hours} where latest is the date of the latest recording, recordings is the number of (1-hr) recordings the speaker appears in, and hours is the total number of hours for that speaker.  Maybe a filter on date range to consider, with the default being the last week.  Id can be a short id, maybe integer or character, ideally colored for likely gender (high voice, possibly female; low voice, possibly male), maybe pink, blue, and purple for unsure.  Can be sorted by any of the columns, default is latest decreasing (most recent at top)  
  * Recordings \- maybe a simple table of {play/pause, MM, date, time, picture, name, conversation}, where   
    * the first is a button that when pressed plays the audio and changes into a pause button, which can be pressed to pause (at which point it turns back into a play button).  If the recording is paused part-way through, pressing play again will resume, maybe a log-press will start from the beginning.    
    * MM is number of minutes duration of the recording, each recording here is a single voice, not a multi-person conversation (unless switched to conversation mode)  
    * Date and time refer to when the recording begins  
    * Picture, name show if a contact is associated, otherwise they are empty, and the name can show the (colorized) speaker id  
    * Conversation could look like a branching-out button, when clicked it would switch the recording area to show all recordings (in time order) within the conversation (which we’ll need to define, but basically a section of time during which one or more people are conversing).  
  * When the page is opened, the top unknown speaker is selected, and their recordings are shown.  The user listens to a few, then the user scrolls/selects the contacts so the right person is shown, and either drags the contact on to the speaker’s recording, or visa-versa, at which point the contact’s voice becomes known (this is how users do the association)

## Diarize

Associating contacts is the first user-deliverable portion of the app.  But since [limitless.ai](http://limitless.ai)’s api doesn’t show speaker id of all unknown speakers (i think it just munges them into Unknown), we’ll probably need to diarize the audio ourselves (which may take more horsepower than my mac has, thus the possibility of doing it on a cloud gpu (we would build the software, using open/available libraries) or using an existing service (eg if such is easy and not too expensive) \[possibly we could use an external service first, and then replace it with a custom cheaper one as part of the experiment\].  My light understanding of such tools is that they identify unique speakers within a given (1 or more person) recording, and then it seems like associating the speaker with a voice model that can be used to identify the person would be an extra stage.  Over time there would be a library of known and unknown speakers, each with an id (as noted earlier).  It seems to me that once a speaker is associated with a contact, all recordings by them would be able to be labeled with their picture and/or name, via a database lookup (rather than having to go back through all the recordings and reprocessing them \- however perhaps my understanding of the tech is wrong.  But i’d think each speaker has a vector and we could somehow suggest most-likely 3 speakers for unknown recordings).

# Visualization

## Calendar

The next user-visible portion of the app would be a page that shows a calendar, with day/week/month options, and known people’s picture appears on the days (maybe sorted by most or least recordings or hours), and you can filter to a set of contacts (eg show me how often i talk with my mom and/or dad).  So there’s probably a contacts list on the page too.

## Timeline

Then I want a timeline, that shows total duration or number of recordings on the y axis, and time or rank on the x axis.  Eg for duration vs time, it’d show how many hours i spoke with who over the time range (selectable, maybe the last week by default), if multiple people are selected (or maybe top N), it’d show as stacked bars per unit time (eg days if week range, morning/afternoon/evening/night if in day range?).  Similarly for recordings vs time (bars now show the number of recordings instead of duration).  And for duration, rank, it’d show who talks the most (to least on the right), or speaks the most often (recordings rank).  If the time range were set to a meeting (eg from integration with calendar), this screen would show who spoke the most often or most time \- useful for giving them feedback, eg you need to give other people more opportunity to speak (eg a quick way to support “you talk too much” with data).

Or more interestingly, select not a contact but a topic or tone, with the same displays.  Eg how negative/positive are my comments (by number or duration)?  Regarding certain people (encouraging or discouraging, love or hate), or topics (eg religion or politics, general or specific eg a certain worldview or candidate etc).

# Personal Relationship Manager (PRM)

Next, and this is a primary goal of the app, is a page where i can see a list of my contacts {picture, name, known}, select one, and see things like how often do i talk with them, about what, what is my tone, what can we infer about what is important to them based on their conversations with me, how they are motivated (internal, external, by relationship or accomplishment, etc).  And a place to add notes (transcribed of course) which is added to the model.  So maybe if i am at a party and see someone across the room, i could quickly look at my phone, click on them, get a reminder of misc data (was their birthday last week, or is it next week?  What open topics/questions do i have with them?  Do i own them money?  Did I tell them I’d do something for them, and have I done it (association with todo list)?
see https://www.nexusfusion.io/en-us/detail/clay/ for a related product we could probably just integrate with.  i got this product idea from the limitless slack channel, someone said they used this but wish it could include limitless data

Maybe i have a stoic practice of reviewing my day and conversations, noting what went well, poorly, how i felt about certain people, how i think they felt (eg maybe i see a trend that tips me off to when ladies might be on their period and i can be more gentle/encouraging/affirming rather than challenging/etc)?  That is transcribed, but more importantly goes into this PRM app where i can quickly be reminded/tipped about those i interact with, and make comments for later (eg after the interaction).  Or maybe I’m looking at my calendar in the morning, who I’ll be interacting with, i can go and review folks in the app, that’ll help me best engage them when i see them (even better \- i don’t have to always check my app, i have reviewed who they are earlier in the day, it’s not that i don’t care about them to know all this stuff without a tool, it’s that my memory and recall latency and bandwidth are limited, so i have this awesome tool i use to help me be a better person in my relationships.  It could be used for bad ends (eg manipulation) or good (eg love one another).

# Goals

A next area for the app is to declare goals, eg i want to be a nicer person, more encouraging, or maybe more challenging/direct with person X.  Or I want to share my religious or political beliefs with other people more often (how often, 1/day?  1/week? etc), or less often.  I can set a goal, and then the app will help me assess my trajectory on that goal.  Eg the timeline screen could show my desired/goal trend, and actual trend.  This gives me visual feedback on my goals.  Eventually I could voice talk with a chatbot about these goals, and adjust my goals, get feedback from it, eg it seems like you’re still struggling to be nice to person X, for instance yesterday you said Y or Z \- how could you have interacted with them differently to be more in-line with this goal of being nicer you have set?

my goal for the app is not to make people more dependent on it, but rather to help them become better people (improved by the ai coach), and also helped (augmented by the app and ai).  So it shouldn’t just splay out tons of words and suggestions to me (like most chatbots do, ask a question in 10 words and get 150 words back), but instead it should be good at helping the person come to understand and perform.  Basically to help them through the four stages of competence [https://en.wikipedia.org/wiki/Four\_stages\_of\_competence](https://en.wikipedia.org/wiki/Four_stages_of_competence)).  What we say and hear is a major factor in our lives, so use the data/recordings to help people become competent in the goals/abilities they set or select eg from lists categorized by religious or political or whatever systems.  Or longer-term, integrate with audible or other audio books, and when someone is listening to a certain self help book (on marriage, parenting, management, etc), dialog with them about it, assessing if they are understanding, retaining, and applying the material.  What do they agree or disagree with, find perplexing, fascinating, etc?  Dialog with them and help them set good goals (SMART, OKR, atomic, etc) to apply the stuff they are reading the book for \- they are reading it for a reason, determine why (direct and indirect elicitation and observation), and help them accomplish the purpose.

Or maybe the app infers from the words they (often?) speak that they are depressed or stressed or whatever about a given topic or person, and the app can suggest books, articles, etc to help, and then help them walk through the content.  Then the app becomes another channel for marketing of content.  While this may provide a high-value marketing channel, I don’t want the app to become a place to be personally bombarded/targetted by people/companies/organizations trying to get them to buy or believe or volunteer or whatever.  So I’d want to be real careful with this part of the app, which is a long way off anyway.

# Other Data

Maybe I bought a Plaud or Bee or other recording device instead of limitless pendant?  But I like the offerings of the above app, so rather than have the app be only a part of limitless, make it a separate service/app, and able to integrate audio data from other services.  Maybe I let those services store/diarize/transcribe/store the recordings, and I just provide another way to access the data, eg I do the analysis, create snappy database and app to interact with it?

Or maybe I bought an always-on camera ([looki.ai](http://looki.ai) or other), that could become another data stream.

And then there are other lifelog data such as eating (I use my fitness pal, my wife uses weight watchers) and movement (i use fitbit, then there are many other options).

Bringing that data into the service, now I can start making associations between activity, food, sleep, entertainment (what did i watch on TV and my recorder picked up?  Or maybe I’ve integrated youtube and amazon prime and netflix watching), my moods (maybe in my daily reviews the ai asks how i felt today, was grumpy, happy, etc, and can help me associate factors (possibly causal)).  Maybe also integrate my locations, eg when i go to place A or B it causes problems (a bar?  A certain church?  A certain club) or brings great joy (same locations different effects).

So the app becomes more like an AI life coach, supported by data, rather than just a PRM, which is just my current interest to do an experiment with to get started on the idea (and experiment with antigravity and beads).  

Related apps/services include [gyroscope.pe](http://gyroscope.pe) and fulcradynamics.  There are many others it seems…

# Personal Landscape

I quit my job in August, have been working \~20hr/wk as independent contractor since (for my former employer only).  I applied for a job at [limitless.ai](http://limitless.ai) three weeks ago but haven’t heard back. I’ve sent an email to my former boss at Sandia National Labs, and expect to interact with him within a week or so.  I like working part-time, and am 55\.  I am unsure if I want a full-time job, or not.  Part of me would like to build this app/service as a way to make a living, but part of me doesn’t \- eg a startup can be very intense and stressful, not sure i want that \- easier to join a team and help them (so i don’t have to lead/boss the whole thing).  On the other hand i like being in control, have experience, etc.  But there is a lot of competition in this area, it seems every app is adding AI coaches.  Maybe this app can leave each to their expertise (eg monarch for money, my fitness pal for food, fitbit for exercise, etc) and instead be the consolidation point, interacting with domain-specific agents.  This seems most likely to succeed.  And I’m interested to learn about MCP etc, and don’t have any experience with it.  So this is another area of learning opportunity for this experiment.

Anyway one idea for experiment is to show [limitless.ai](http://limitless.ai) that i have skills and ideas and vision, eg “hey check out this app i wrote” \- if they like it perhaps they’d be more interested in hiring me, or i could be a development partner with them (but on what income stream for my living expenses)?  Etc.  so that is wider personal context for the app.  So I’m thinking maybe expose a public github with info sufficient to show, but not show all these personal comments or the full vision for the app, so maybe track that stuff in a fork?  Or visa-versa (private app is the big overall vision and personal bits, and public app is the subset that i can show others, and maybe get collaborators on etc)?  I welcome your feedback/advice on all of this, app, programing, architecture, stack, feasibility, cost, timeline, competition, value proposition, github practices, etc.

# Misc

There are probably multiple repos here, eg one for [limitless.ai](http://limitless.ai) data getting/storage, one for diarization, one for the user app.  Some public, some private.  I wonder how well ai ide’s and beads are able to work across multiple repositories…  I am thinking, a main private repo called hai (for holy ai, like the holy spirit, but ai) for the app, and then at least one other repo for getting (and serving) data (eg same shape as limitless but add the speaker id on recordings), maybe another for diarization.  
limitless \-\> data/db/api \-\> {diarize, hai}

seed.jpg shows some relevant whiteboard stuff.  it includes a little section on hierarchical grouping of events.  
eg come up with some general data structure, that can be composed recursively, eg an event might be of type 
single_speaker, or multi_speaker (or conversation).  ideally the thing is connected to user calendar, so if
there is a meeting with two people, and one's voice is known, hai can ask if the other voice is the other meeting
participant (boy, that was easy!).  this seems like a topic for later
