wrk.path  = "/"
wrk.method = "POST"
wrk.body   = "id=7579764573763278289&platformid=ANDROID&version=1.0.0"
wrk.headers["Content-Type"] = "application/x-www-form-urlencoded"
wrk.headers["deviceid"] = "354435052821931"
wrk.headers["Accept"] = "application/json"


local counter = 1
local threads = {}

function setup(thread)
   thread:set("id", counter)
   table.insert(threads, thread)
   counter = counter + 1
end

function init(args)
  st = {}
end


function response(status, headers, body)
  if st[status] == nil then
     st[status] = 0
  end
  
  st[status] = st[status] + 1
end


function done(summary, latency, requests)
  for index, thread in ipairs(threads) do
    print("**********************************")
    local ss =  thread:get("st")
    for s, n in pairs(ss) do
      msg = "%d: %d"
      print(msg:format(n,s))
    end
  end
end

