local sessionKey = KEYS[1]
local ip = ARGV[1]
local vote = tonumber(ARGV[2])

local avg = tonumber(redis.call('HGET', sessionKey, 'avg') or "0")
local numVotes = tonumber(redis.call('HGET', sessionKey, 'numVotes') or "0")
local ipExists = redis.call('HEXISTS', sessionKey, ip)

if ipExists == 1 then
    local oldVote = tonumber(redis.call('HGET', sessionKey, ip))
    avg = (avg * numVotes - oldVote + vote) / numVotes
else
    redis.call('HINCRBY', sessionKey, 'numVotes', 1)
    avg = (avg * numVotes + vote) / (numVotes + 1)
    numVotes = numVotes + 1
end

redis.call('HSET', sessionKey, 'avg', tostring(avg))
redis.call('HSET', sessionKey, ip, tostring(vote))

return {tostring(avg), tostring(numVotes)}

